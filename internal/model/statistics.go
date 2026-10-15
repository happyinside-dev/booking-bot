package model

type BusinessStatistics struct {
	Total, Upcoming, Completed, Cancelled, NoShow int64
	CancellationRate                              float64
	PotentialRevenue, ActualRevenue               float64
	PopularServices                               []ServiceStatistics
	EmployeeLoad                                  []EmployeeStatistics
}

type ServiceStatistics struct {
	ServiceID int64
	Name      string
	Bookings  int64
	Revenue   float64
}

type EmployeeStatistics struct {
	EmployeeID       int64
	Name             string
	Bookings         int64
	Cancelled        int64
	BookedMinutes    int64
	PotentialRevenue float64
}
