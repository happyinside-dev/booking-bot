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
	flowServiceCreate  = "service_create"
	flowServiceUpdate  = "service_update"
	flowServiceStatus  = "service_status"
	flowEmployeeCreate = "employee_create"
	flowEmployeeUpdate = "employee_update"
	flowEmployeeStatus = "employee_status"
	flowSchedule       = "schedule"
	flowBlocked        = "blocked"
)

type adminContext struct {
	user     *model.User
	business *model.Business
}

func (h *Handler) adminContext(ctx context.Context, telegramID int64) (*adminContext, error) {
	user, err := h.userService.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, err
	}
	businessID := h.botContext.BusinessID
	if err := h.businessService.HasRole(ctx, businessID, user.ID, model.RoleOwner, model.RoleAdmin); err != nil {
		return nil, err
	}
	business, err := h.businessService.GetByID(ctx, businessID)
	if err != nil {
		return nil, err
	}
	return &adminContext{user: user, business: business}, nil
}

func (h *Handler) ServicesMenu(ctx context.Context, b *bot.Bot, update *models.Update) {
	admin, ok := h.requireAdmin(ctx, b, update)
	if !ok {
		return
	}
	items, err := h.catalogService.List(ctx, admin.user.ID, admin.business.ID)
	if err != nil {
		h.adminError(ctx, b, update.Message.Chat.ID, err)
		return
	}
	var out strings.Builder
	out.WriteString("💈 Услуги\n")
	if len(items) == 0 {
		out.WriteString("Список пока пуст.")
	}
	for _, item := range items {
		status := "активна"
		if !item.IsActive {
			status = "неактивна"
		}
		fmt.Fprintf(&out, "\n#%d — %s, %d мин., %.2f ₽ (%s)", item.ID, item.Name, item.DurationMin, item.Price, status)
	}
	h.sendWithKeyboard(ctx, b, update.Message.Chat.ID, out.String(), keyboards.ServiceAdminKeyboard())
}

func (h *Handler) EmployeesMenu(ctx context.Context, b *bot.Bot, update *models.Update) {
	admin, ok := h.requireAdmin(ctx, b, update)
	if !ok {
		return
	}
	items, err := h.employeeService.List(ctx, admin.user.ID, admin.business.ID)
	if err != nil {
		h.adminError(ctx, b, update.Message.Chat.ID, err)
		return
	}
	var out strings.Builder
	out.WriteString("👨‍💼 Сотрудники\n")
	if len(items) == 0 {
		out.WriteString("Список пока пуст.")
	}
	for _, item := range items {
		status := "активен"
		if !item.IsActive {
			status = "неактивен"
		}
		fmt.Fprintf(&out, "\n#%d — %s (%s)", item.ID, item.Name, status)
	}
	h.sendWithKeyboard(ctx, b, update.Message.Chat.ID, out.String(), keyboards.EmployeeAdminKeyboard())
}

func (h *Handler) BackToAdmin(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}
	_ = h.states.Delete(ctx, update.Message.From.ID)
	h.Admin(ctx, b, update)
}

func (h *Handler) Cancel(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}
	_ = h.states.Delete(ctx, update.Message.From.ID)
	h.sendWithKeyboard(ctx, b, update.Message.Chat.ID, "Действие отменено.", keyboards.AdminKeyboard())
}

func (h *Handler) BeginServiceCreate(ctx context.Context, b *bot.Bot, u *models.Update) {
	h.begin(ctx, b, u, flowServiceCreate, "name", "Введите название услуги:")
}
func (h *Handler) BeginServiceUpdate(ctx context.Context, b *bot.Bot, u *models.Update) {
	h.begin(ctx, b, u, flowServiceUpdate, "id", "Введите ID услуги:")
}
func (h *Handler) BeginServiceStatus(ctx context.Context, b *bot.Bot, u *models.Update) {
	h.begin(ctx, b, u, flowServiceStatus, "id", "Введите ID услуги:")
}
func (h *Handler) BeginEmployeeCreate(ctx context.Context, b *bot.Bot, u *models.Update) {
	h.begin(ctx, b, u, flowEmployeeCreate, "name", "Введите имя сотрудника:")
}
func (h *Handler) BeginEmployeeUpdate(ctx context.Context, b *bot.Bot, u *models.Update) {
	h.begin(ctx, b, u, flowEmployeeUpdate, "id", "Введите ID сотрудника:")
}
func (h *Handler) BeginEmployeeStatus(ctx context.Context, b *bot.Bot, u *models.Update) {
	h.begin(ctx, b, u, flowEmployeeStatus, "id", "Введите ID сотрудника:")
}
func (h *Handler) BeginSchedule(ctx context.Context, b *bot.Bot, u *models.Update) {
	h.begin(ctx, b, u, flowSchedule, "employee_id", "Введите ID сотрудника:")
}
func (h *Handler) BeginBlocked(ctx context.Context, b *bot.Bot, u *models.Update) {
	h.begin(ctx, b, u, flowBlocked, "employee_id", "Введите ID сотрудника:")
}

func (h *Handler) begin(ctx context.Context, b *bot.Bot, update *models.Update, flow, step, prompt string) {
	if _, ok := h.requireAdmin(ctx, b, update); !ok {
		return
	}
	state := &fsm.State{Flow: flow, Step: step, Data: map[string]string{}}
	if err := h.states.Set(ctx, update.Message.From.ID, state); err != nil {
		h.adminError(ctx, b, update.Message.Chat.ID, err)
		return
	}
	h.sendWithKeyboard(ctx, b, update.Message.Chat.ID, prompt, keyboards.CancelKeyboard())
}

func (h *Handler) handleState(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	state, err := h.states.Get(ctx, update.Message.From.ID)
	if errors.Is(err, fsm.ErrNotFound) {
		return false
	}
	if err != nil {
		h.adminError(ctx, b, update.Message.Chat.ID, err)
		return true
	}
	admin, err := h.adminContext(ctx, update.Message.From.ID)
	if err != nil {
		_ = h.states.Delete(ctx, update.Message.From.ID)
		h.adminError(ctx, b, update.Message.Chat.ID, err)
		return true
	}
	text := strings.TrimSpace(update.Message.Text)
	var prompt string
	var done bool
	switch state.Flow {
	case flowServiceCreate, flowServiceUpdate:
		prompt, done, err = h.serviceForm(ctx, state, admin, text)
	case flowServiceStatus:
		prompt, done, err = h.serviceStatus(ctx, state, admin, text)
	case flowEmployeeCreate, flowEmployeeUpdate:
		prompt, done, err = h.employeeForm(ctx, state, admin, text)
	case flowEmployeeStatus:
		prompt, done, err = h.employeeStatus(ctx, state, admin, text)
	case flowSchedule:
		prompt, done, err = h.scheduleForm(ctx, state, admin, text)
	case flowBlocked:
		prompt, done, err = h.blockedForm(ctx, state, admin, text)
	default:
		_ = h.states.Delete(ctx, update.Message.From.ID)
		return false
	}
	if err != nil {
		h.sendText(ctx, b, update.Message.Chat.ID, humanAdminError(err))
		return true
	}
	if done {
		_ = h.states.Delete(ctx, update.Message.From.ID)
		h.sendWithKeyboard(ctx, b, update.Message.Chat.ID, prompt, keyboards.AdminKeyboard())
		return true
	}
	if err := h.states.Set(ctx, update.Message.From.ID, state); err != nil {
		h.adminError(ctx, b, update.Message.Chat.ID, err)
		return true
	}
	h.sendWithKeyboard(ctx, b, update.Message.Chat.ID, prompt, keyboards.CancelKeyboard())
	return true
}

func (h *Handler) serviceForm(ctx context.Context, st *fsm.State, a *adminContext, text string) (string, bool, error) {
	switch st.Step {
	case "id":
		if _, err := positiveInt64(text); err != nil {
			return "", false, err
		}
		st.Data["id"] = text
		st.Step = "name"
		return "Введите новое название:", false, nil
	case "name":
		if text == "" {
			return "", false, apperr.ErrInvalidInput
		}
		st.Data["name"] = text
		st.Step = "description"
		return "Введите описание (или -):", false, nil
	case "description":
		if text == "-" {
			text = ""
		}
		st.Data["description"] = text
		st.Step = "duration"
		return "Введите длительность в минутах:", false, nil
	case "duration":
		if _, err := positiveInt(text); err != nil {
			return "", false, err
		}
		st.Data["duration"] = text
		st.Step = "price"
		return "Введите цену, например 1500 или 1500.50:", false, nil
	case "price":
		price, err := nonNegativeFloat(text)
		if err != nil {
			return "", false, err
		}
		duration, _ := strconv.Atoi(st.Data["duration"])
		if st.Flow == flowServiceCreate {
			_, err = h.catalogService.Create(ctx, service.CreateServiceInput{ActorUserID: a.user.ID, BusinessID: a.business.ID, Name: st.Data["name"], Description: st.Data["description"], DurationMin: duration, Price: price})
		} else {
			id, _ := strconv.ParseInt(st.Data["id"], 10, 64)
			_, err = h.catalogService.Update(ctx, service.UpdateServiceInput{ActorUserID: a.user.ID, BusinessID: a.business.ID, ServiceID: id, Name: st.Data["name"], Description: st.Data["description"], DurationMin: duration, Price: price})
		}
		if err != nil {
			return "", false, err
		}
		return "✅ Услуга сохранена.", true, nil
	}
	return "", false, apperr.ErrInvalidInput
}

func (h *Handler) serviceStatus(ctx context.Context, st *fsm.State, a *adminContext, text string) (string, bool, error) {
	if st.Step == "id" {
		if _, err := positiveInt64(text); err != nil {
			return "", false, err
		}
		st.Data["id"] = text
		st.Step = "active"
		return "Введите «вкл» или «выкл»:", false, nil
	}
	active, err := parseActive(text)
	if err != nil {
		return "", false, err
	}
	id, _ := strconv.ParseInt(st.Data["id"], 10, 64)
	if err := h.catalogService.SetActive(ctx, a.user.ID, a.business.ID, id, active); err != nil {
		return "", false, err
	}
	return "✅ Статус услуги обновлён.", true, nil
}

func (h *Handler) employeeForm(ctx context.Context, st *fsm.State, a *adminContext, text string) (string, bool, error) {
	switch st.Step {
	case "id":
		if _, err := positiveInt64(text); err != nil {
			return "", false, err
		}
		st.Data["id"] = text
		st.Step = "name"
		return "Введите новое имя:", false, nil
	case "name":
		if text == "" {
			return "", false, apperr.ErrInvalidInput
		}
		st.Data["name"] = text
		st.Step = "description"
		return "Введите описание (или -):", false, nil
	case "description":
		if text == "-" {
			text = ""
		}
		var err error
		if st.Flow == flowEmployeeCreate {
			_, err = h.employeeService.Create(ctx, service.CreateEmployeeInput{ActorUserID: a.user.ID, BusinessID: a.business.ID, Name: st.Data["name"], Description: text})
		} else {
			id, _ := strconv.ParseInt(st.Data["id"], 10, 64)
			_, err = h.employeeService.Update(ctx, service.UpdateEmployeeInput{ActorUserID: a.user.ID, BusinessID: a.business.ID, EmployeeID: id, Name: st.Data["name"], Description: text})
		}
		if err != nil {
			return "", false, err
		}
		return "✅ Сотрудник сохранён.", true, nil
	}
	return "", false, apperr.ErrInvalidInput
}

func (h *Handler) employeeStatus(ctx context.Context, st *fsm.State, a *adminContext, text string) (string, bool, error) {
	if st.Step == "id" {
		if _, err := positiveInt64(text); err != nil {
			return "", false, err
		}
		st.Data["id"] = text
		st.Step = "active"
		return "Введите «вкл» или «выкл»:", false, nil
	}
	active, err := parseActive(text)
	if err != nil {
		return "", false, err
	}
	id, _ := strconv.ParseInt(st.Data["id"], 10, 64)
	if err := h.employeeService.SetActive(ctx, a.user.ID, a.business.ID, id, active); err != nil {
		return "", false, err
	}
	return "✅ Статус сотрудника обновлён.", true, nil
}

func (h *Handler) scheduleForm(ctx context.Context, st *fsm.State, a *adminContext, text string) (string, bool, error) {
	switch st.Step {
	case "employee_id":
		if _, err := positiveInt64(text); err != nil {
			return "", false, err
		}
		st.Data["employee_id"] = text
		st.Step = "weekday"
		return "Введите день недели (1 — Пн, …, 7 — Вс):", false, nil
	case "weekday":
		day, err := strconv.Atoi(text)
		if err != nil || day < 1 || day > 7 {
			return "", false, apperr.ErrInvalidInput
		}
		st.Data["weekday"] = text
		st.Step = "start"
		return "Введите начало HH:MM или «выходной»:", false, nil
	case "start":
		if strings.EqualFold(text, "выходной") {
			return h.saveSchedule(ctx, st, a, 0, 0, true)
		}
		minute, err := parseClock(text)
		if err != nil {
			return "", false, err
		}
		st.Data["start"] = strconv.Itoa(minute)
		st.Step = "end"
		return "Введите окончание HH:MM:", false, nil
	case "end":
		minute, err := parseClock(text)
		if err != nil {
			return "", false, err
		}
		start, _ := strconv.Atoi(st.Data["start"])
		return h.saveSchedule(ctx, st, a, start, minute, false)
	}
	return "", false, apperr.ErrInvalidInput
}

func (h *Handler) saveSchedule(ctx context.Context, st *fsm.State, a *adminContext, start, end int, dayOff bool) (string, bool, error) {
	employeeID, _ := strconv.ParseInt(st.Data["employee_id"], 10, 64)
	weekday, _ := strconv.Atoi(st.Data["weekday"])
	_, err := h.scheduleService.SetWorkingHours(ctx, service.SetWorkingHoursInput{ActorUserID: a.user.ID, BusinessID: a.business.ID, EmployeeID: employeeID, Weekday: weekday, StartMinute: start, EndMinute: end, IsDayOff: dayOff})
	if err != nil {
		return "", false, err
	}
	return "✅ Расписание сохранено.", true, nil
}

func (h *Handler) blockedForm(ctx context.Context, st *fsm.State, a *adminContext, text string) (string, bool, error) {
	switch st.Step {
	case "employee_id":
		if _, err := positiveInt64(text); err != nil {
			return "", false, err
		}
		st.Data["employee_id"] = text
		st.Step = "date"
		return "Введите дату в формате ДД.ММ.ГГГГ:", false, nil
	case "date":
		if _, err := time.Parse("02.01.2006", text); err != nil {
			return "", false, apperr.ErrInvalidInput
		}
		st.Data["date"] = text
		st.Step = "start"
		return "Введите начало HH:MM:", false, nil
	case "start":
		if _, err := parseClock(text); err != nil {
			return "", false, err
		}
		st.Data["start"] = text
		st.Step = "end"
		return "Введите окончание HH:MM:", false, nil
	case "end":
		if _, err := parseClock(text); err != nil {
			return "", false, err
		}
		st.Data["end"] = text
		st.Step = "reason"
		return "Введите причину (или -):", false, nil
	case "reason":
		loc, err := time.LoadLocation(a.business.Timezone)
		if err != nil {
			return "", false, apperr.ErrInvalidTimezone
		}
		start, err := time.ParseInLocation("02.01.2006 15:04", st.Data["date"]+" "+st.Data["start"], loc)
		if err != nil {
			return "", false, apperr.ErrInvalidInput
		}
		end, err := time.ParseInLocation("02.01.2006 15:04", st.Data["date"]+" "+st.Data["end"], loc)
		if err != nil {
			return "", false, apperr.ErrInvalidInput
		}
		if text == "-" {
			text = ""
		}
		employeeID, _ := strconv.ParseInt(st.Data["employee_id"], 10, 64)
		_, err = h.scheduleService.CreateBlockedSlot(ctx, service.CreateBlockedSlotInput{ActorUserID: a.user.ID, BusinessID: a.business.ID, EmployeeID: employeeID, StartAt: start, EndAt: end, Reason: text})
		if err != nil {
			return "", false, err
		}
		return "✅ Время заблокировано.", true, nil
	}
	return "", false, apperr.ErrInvalidInput
}

func (h *Handler) requireAdmin(ctx context.Context, b *bot.Bot, update *models.Update) (*adminContext, bool) {
	if update.Message == nil || update.Message.From == nil {
		return nil, false
	}
	a, err := h.adminContext(ctx, update.Message.From.ID)
	if err != nil {
		h.adminError(ctx, b, update.Message.Chat.ID, err)
		return nil, false
	}
	return a, true
}

func (h *Handler) adminError(ctx context.Context, b *bot.Bot, chatID int64, err error) {
	h.sendText(ctx, b, chatID, humanAdminError(err))
}
func humanAdminError(err error) string {
	switch {
	case errors.Is(err, apperr.ErrForbidden):
		return "❌ У вас нет доступа к этому действию."
	case errors.Is(err, apperr.ErrNotFound):
		return "❌ Объект не найден."
	case errors.Is(err, apperr.ErrInvalidInput), errors.Is(err, apperr.ErrInvalidSchedule):
		return "❌ Некорректное значение. Повторите ввод или нажмите «❌ Отмена»."
	default:
		return "Произошла ошибка, попробуйте ещё раз позже."
	}
}

func (h *Handler) sendWithKeyboard(ctx context.Context, b *bot.Bot, chatID int64, text string, keyboard models.ReplyKeyboardMarkup) {
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: text, ReplyMarkup: keyboard}); err != nil {
		h.log.Error("failed to send admin message", "error", err)
	}
}

func positiveInt64(value string) (int64, error) {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil || v <= 0 {
		return 0, apperr.ErrInvalidInput
	}
	return v, nil
}
func positiveInt(value string) (int, error) {
	v, err := strconv.Atoi(value)
	if err != nil || v <= 0 {
		return 0, apperr.ErrInvalidInput
	}
	return v, nil
}
func nonNegativeFloat(value string) (float64, error) {
	value = strings.ReplaceAll(value, ",", ".")
	v, err := strconv.ParseFloat(value, 64)
	if err != nil || v < 0 {
		return 0, apperr.ErrInvalidInput
	}
	return v, nil
}
func parseClock(value string) (int, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, apperr.ErrInvalidInput
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}
func parseActive(value string) (bool, error) {
	switch strings.ToLower(value) {
	case "вкл", "активировать":
		return true, nil
	case "выкл", "деактивировать":
		return false, nil
	}
	return false, apperr.ErrInvalidInput
}
