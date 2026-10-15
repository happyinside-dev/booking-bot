package handlers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/happyinside-dev/booking-bot/internal/apperr"
	"github.com/happyinside-dev/booking-bot/internal/bot/fsm"
	"github.com/happyinside-dev/booking-bot/internal/bot/keyboards"
	"github.com/happyinside-dev/booking-bot/internal/model"
	"github.com/happyinside-dev/booking-bot/internal/service"
)

const (
	flowBooking       = "booking"
	flowBookingCancel = "booking_cancel"
)

func (h *Handler) BeginBooking(ctx context.Context, b *bot.Bot, u *models.Update) {
	if u.Message == nil || u.Message.From == nil {
		return
	}
	if _, err := h.userService.GetByTelegramID(ctx, u.Message.From.ID); err != nil {
		h.sendText(ctx, b, u.Message.Chat.ID, "Сначала отправьте /start.")
		return
	}
	items, err := h.bookingService.ListServices(ctx, h.botContext.BusinessID)
	if err != nil {
		h.adminError(ctx, b, u.Message.Chat.ID, err)
		return
	}
	if len(items) == 0 {
		h.sendText(ctx, b, u.Message.Chat.ID, "Сейчас нет доступных услуг.")
		return
	}
	var out strings.Builder
	out.WriteString("Выберите услугу и отправьте её ID:\n")
	for _, v := range items {
		fmt.Fprintf(&out, "\n#%d — %s, %d мин., %.2f ₽", v.ID, v.Name, v.DurationMin, v.Price)
	}
	if err := h.states.Set(ctx, u.Message.From.ID, &fsm.State{Flow: flowBooking, Step: "service", Data: map[string]string{}}); err != nil {
		h.adminError(ctx, b, u.Message.Chat.ID, err)
		return
	}
	h.sendWithKeyboard(ctx, b, u.Message.Chat.ID, out.String(), keyboards.CancelKeyboard())
}

func (h *Handler) MyBookings(ctx context.Context, b *bot.Bot, u *models.Update) {
	if u.Message == nil || u.Message.From == nil {
		return
	}
	items, err := h.bookingService.ListMine(ctx, u.Message.From.ID, h.botContext.BusinessID)
	if err != nil {
		h.adminError(ctx, b, u.Message.Chat.ID, err)
		return
	}
	h.sendWithKeyboard(ctx, b, u.Message.Chat.ID, formatBookings(items), keyboards.MyBookingsKeyboard())
}

func (h *Handler) BeginBookingCancel(ctx context.Context, b *bot.Bot, u *models.Update) {
	h.beginClient(ctx, b, u, flowBookingCancel, "id", "Введите ID записи для отмены:")
}
func (h *Handler) MainMenu(ctx context.Context, b *bot.Bot, u *models.Update) {
	if u.Message == nil {
		return
	}
	if u.Message.From != nil {
		_ = h.states.Delete(ctx, u.Message.From.ID)
	}
	h.sendWithKeyboard(ctx, b, u.Message.Chat.ID, "Главное меню", keyboards.MainMenu())
}

func (h *Handler) AdminToday(ctx context.Context, b *bot.Bot, u *models.Update) {
	h.adminBookings(ctx, b, u, true)
}
func (h *Handler) AdminAll(ctx context.Context, b *bot.Bot, u *models.Update) {
	h.adminBookings(ctx, b, u, false)
}
func (h *Handler) adminBookings(ctx context.Context, b *bot.Bot, u *models.Update, todayOnly bool) {
	a, ok := h.requireAdmin(ctx, b, u)
	if !ok {
		return
	}
	items, err := h.bookingService.ListBusiness(ctx, a.user.ID, a.business.ID, h.businessService)
	if err != nil {
		h.adminError(ctx, b, u.Message.Chat.ID, err)
		return
	}
	if todayOnly {
		loc, _ := time.LoadLocation(a.business.Timezone)
		now := time.Now().In(loc)
		filtered := items[:0]
		for _, v := range items {
			local := v.StartAt.In(loc)
			if local.Year() == now.Year() && local.YearDay() == now.YearDay() {
				filtered = append(filtered, v)
			}
		}
		items = filtered
	}
	h.sendWithKeyboard(ctx, b, u.Message.Chat.ID, formatBookings(items), keyboards.AdminKeyboard())
}

func (h *Handler) beginClient(ctx context.Context, b *bot.Bot, u *models.Update, flow, step, prompt string) {
	if u.Message == nil || u.Message.From == nil {
		return
	}
	if _, err := h.userService.GetByTelegramID(ctx, u.Message.From.ID); err != nil {
		h.sendText(ctx, b, u.Message.Chat.ID, "Сначала отправьте /start.")
		return
	}
	if err := h.states.Set(ctx, u.Message.From.ID, &fsm.State{Flow: flow, Step: step, Data: map[string]string{}}); err != nil {
		h.adminError(ctx, b, u.Message.Chat.ID, err)
		return
	}
	h.sendWithKeyboard(ctx, b, u.Message.Chat.ID, prompt, keyboards.CancelKeyboard())
}

func (h *Handler) bookingForm(ctx context.Context, st *fsm.State, business *model.Business, telegramID int64, text string) (string, bool, error) {
	switch st.Flow {
	case flowBookingCancel:
		id, err := positiveInt64(text)
		if err != nil {
			return "", false, err
		}
		if err := h.bookingService.Cancel(ctx, telegramID, id); err != nil {
			return "", false, err
		}
		return "✅ Запись отменена.", true, nil
	}
	switch st.Step {
	case "service":
		id, err := positiveInt64(text)
		if err != nil {
			return "", false, err
		}
		items, err := h.bookingService.ListServices(ctx, business.ID)
		if err != nil {
			return "", false, err
		}
		found := false
		for _, v := range items {
			if v.ID == id {
				found = true
				st.Data["service_id"] = text
				st.Data["service_name"] = v.Name
				st.Data["price"] = fmt.Sprintf("%.2f", v.Price)
				st.Data["duration"] = strconv.Itoa(v.DurationMin)
			}
		}
		if !found {
			return "", false, apperr.ErrNotFound
		}
		employees, err := h.bookingService.ListEmployees(ctx, business.ID)
		if err != nil {
			return "", false, err
		}
		if len(employees) == 0 {
			return "", false, apperr.ErrNotFound
		}
		if len(employees) == 1 {
			st.Data["employee_id"] = strconv.FormatInt(employees[0].ID, 10)
			st.Data["employee_name"] = employees[0].Name
			st.Step = "date"
			return "Введите дату ДД.ММ.ГГГГ:", false, nil
		}
		var out strings.Builder
		out.WriteString("Выберите сотрудника и отправьте ID:\n")
		for _, v := range employees {
			fmt.Fprintf(&out, "\n#%d — %s", v.ID, v.Name)
		}
		st.Step = "employee"
		return out.String(), false, nil
	case "employee":
		id, err := positiveInt64(text)
		if err != nil {
			return "", false, err
		}
		items, err := h.bookingService.ListEmployees(ctx, business.ID)
		if err != nil {
			return "", false, err
		}
		found := false
		for _, v := range items {
			if v.ID == id {
				found = true
				st.Data["employee_id"] = text
				st.Data["employee_name"] = v.Name
			}
		}
		if !found {
			return "", false, apperr.ErrNotFound
		}
		st.Step = "date"
		return "Введите дату ДД.ММ.ГГГГ:", false, nil
	case "date":
		loc, err := time.LoadLocation(business.Timezone)
		if err != nil {
			return "", false, apperr.ErrInvalidTimezone
		}
		date, err := time.ParseInLocation("02.01.2006", text, loc)
		if err != nil {
			return "", false, apperr.ErrInvalidInput
		}
		employeeID, _ := strconv.ParseInt(st.Data["employee_id"], 10, 64)
		serviceID, _ := strconv.ParseInt(st.Data["service_id"], 10, 64)
		slots, err := h.bookingService.AvailableSlots(ctx, employeeID, serviceID, date)
		if err != nil {
			return "", false, err
		}
		if len(slots) == 0 {
			return "На эту дату свободного времени нет. Введите другую дату:", false, nil
		}
		st.Data["date"] = text
		var out strings.Builder
		out.WriteString("Введите доступное время HH:MM:\n")
		var encoded []string
		for _, slot := range slots {
			local := slot.In(loc)
			fmt.Fprintf(&out, "\n%s", local.Format("15:04"))
			encoded = append(encoded, slot.UTC().Format(time.RFC3339))
		}
		st.Data["slots"] = strings.Join(encoded, ",")
		st.Step = "time"
		return out.String(), false, nil
	case "time":
		loc, _ := time.LoadLocation(business.Timezone)
		clock, err := time.Parse("15:04", text)
		if err != nil {
			return "", false, apperr.ErrInvalidInput
		}
		date, _ := time.ParseInLocation("02.01.2006", st.Data["date"], loc)
		chosen := time.Date(date.Year(), date.Month(), date.Day(), clock.Hour(), clock.Minute(), 0, 0, loc).UTC()
		valid := false
		for _, raw := range strings.Split(st.Data["slots"], ",") {
			slot, _ := time.Parse(time.RFC3339, raw)
			if slot.Equal(chosen) {
				valid = true
			}
		}
		if !valid {
			return "", false, apperr.ErrInvalidTime
		}
		st.Data["start_at"] = chosen.Format(time.RFC3339)
		st.Step = "confirm"
		return fmt.Sprintf("📋 Ваша запись\n\nУслуга: %s\nМастер: %s\nДата: %s\nВремя: %s\nДлительность: %s минут\nСтоимость: %s ₽\n\nОтправьте «✅ Подтвердить запись».", st.Data["service_name"], st.Data["employee_name"], st.Data["date"], text, st.Data["duration"], st.Data["price"]), false, nil
	case "confirm":
		if text != "✅ Подтвердить запись" {
			return "", false, apperr.ErrInvalidInput
		}
		employeeID, _ := strconv.ParseInt(st.Data["employee_id"], 10, 64)
		serviceID, _ := strconv.ParseInt(st.Data["service_id"], 10, 64)
		start, _ := time.Parse(time.RFC3339, st.Data["start_at"])
		_, err := h.bookingService.Create(ctx, service.CreateBookingInput{BusinessID: business.ID, EmployeeID: employeeID, ServiceID: serviceID, TelegramID: telegramID, StartAt: start})
		if err != nil {
			return "", false, err
		}
		return "✅ Вы записаны!", true, nil
	}
	return "", false, apperr.ErrInvalidInput
}

func formatBookings(items []model.BookingView) string {
	if len(items) == 0 {
		return "Записей пока нет."
	}
	var out strings.Builder
	out.WriteString("📋 Записи\n")
	for _, v := range items {
		loc, err := time.LoadLocation(v.BusinessTimezone)
		if err != nil {
			loc = time.UTC
		}
		local := v.StartAt.In(loc)
		fmt.Fprintf(&out, "\n#%d · %s\n👨 %s\n📅 %s %s\n💰 %.2f ₽ · %s\n", v.ID, v.ServiceName, v.EmployeeName, local.Format("02.01.2006"), local.Format("15:04"), v.Price, v.Status)
	}
	return out.String()
}

func bookingError(err error) string {
	switch {
	case errors.Is(err, apperr.ErrBookingConflict):
		return "Это время только что заняли. Выберите другое."
	case errors.Is(err, apperr.ErrInvalidTime):
		return "Выбранное время недоступно."
	case errors.Is(err, apperr.ErrBookingAlreadyCancelled):
		return "Эта запись уже отменена."
	case errors.Is(err, apperr.ErrBookingAlreadyStarted):
		return "Начавшуюся или завершённую запись отменить нельзя."
	case errors.Is(err, apperr.ErrNotFound):
		return "Запись не найдена."
	}
	return humanAdminError(err)
}
