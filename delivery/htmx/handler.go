package htmx

import (
	"fmt"
	"html/template"
	"net/http"

	_interface "jobqueue/interface"

	"github.com/labstack/echo/v4"
)

type DashboardHandler struct {
	jobService _interface.JobService
}

func NewDashboardHandler(jobService _interface.JobService) *DashboardHandler {
	return &DashboardHandler{jobService: jobService}
}

func (h *DashboardHandler) Page(c echo.Context) error {
	tmpl, err := template.ParseFiles("./web/htmx/dashboard.html")
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return tmpl.Execute(c.Response().Writer, nil)
}

func (h *DashboardHandler) Message(c echo.Context) error {
	return c.HTML(http.StatusOK, `<p id="message">Welcome to the Job Queue Dashboard. Status and jobs auto-refresh below.</p>`)
}

func (h *DashboardHandler) StatusFragment(c echo.Context) error {
	status, err := h.jobService.GetJobStatus(c.Request().Context())
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	html := fmt.Sprintf(
		`<div id="status-summary" hx-get="/jobqueue/dashboard/status" hx-trigger="every 2s" hx-swap="outerHTML" class="badges">
			<div class="badge pending">Pending<br/>%d</div>
			<div class="badge running">Running<br/>%d</div>
			<div class="badge failed">Failed<br/>%d</div>
			<div class="badge completed">Completed<br/>%d</div>
		</div>`,
		status.Pending, status.Running, status.Failed, status.Completed,
	)
	return c.HTML(http.StatusOK, html)
}

func (h *DashboardHandler) JobsFragment(c echo.Context) error {
	jobs, err := h.jobService.GetAllJobs(c.Request().Context())
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	html := `<div id="jobs-table" hx-get="/jobqueue/dashboard/jobs" hx-trigger="every 2s" hx-swap="outerHTML"><table>
		<thead><tr><th>ID</th><th>Task</th><th>Status</th><th>Attempts</th><th>View</th></tr></thead>
		<tbody>`
	if len(jobs) == 0 {
		html += `<tr><td colspan="5">No jobs yet.</td></tr>`
	} else {
		for _, j := range jobs {
			html += fmt.Sprintf(
				`<tr><td>%s</td><td>%s</td><td>%s</td><td>%d</td>
				<td><button class="view-btn" hx-get="/jobqueue/dashboard/jobs/%s" hx-target="#job-detail" hx-swap="outerHTML">View</button></td></tr>`,
				template.HTMLEscapeString(j.ID),
				template.HTMLEscapeString(j.Task),
				template.HTMLEscapeString(j.Status),
				j.Attempts,
				template.HTMLEscapeString(j.ID),
			)
		}
	}
	html += `</tbody></table></div>`
	return c.HTML(http.StatusOK, html)
}

func (h *DashboardHandler) renderJobDetail(id string, ctx echo.Context) string {
	job, err := h.jobService.FindByID(ctx.Request().Context(), id)
	if err != nil {
		return fmt.Sprintf(`<div id="job-detail">Job %s not found.</div>`, template.HTMLEscapeString(id))
	}
	return fmt.Sprintf(
		`<div id="job-detail">
			<strong>ID:</strong> %s<br/>
			<strong>Task:</strong> %s<br/>
			<strong>Status:</strong> %s<br/>
			<strong>Attempts:</strong> %d
		</div>`,
		template.HTMLEscapeString(job.ID),
		template.HTMLEscapeString(job.Task),
		template.HTMLEscapeString(job.Status),
		job.Attempts,
	)
}

func (h *DashboardHandler) JobDetailFragment(c echo.Context) error {
	return c.HTML(http.StatusOK, h.renderJobDetail(c.Param("id"), c))
}

func (h *DashboardHandler) JobDetailByQuery(c echo.Context) error {
	return c.HTML(http.StatusOK, h.renderJobDetail(c.QueryParam("id"), c))
}

func (h *DashboardHandler) CreateJobs(c echo.Context) error {
	ctx := c.Request().Context()
	jobs := []string{
		c.FormValue("Job1"),
		c.FormValue("Job2"),
		c.FormValue("Job3"),
	}
	for _, t := range jobs {
		if t == "" {
			continue
		}
		if _, err := h.jobService.Enqueue(ctx, t); err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
	}
	return h.JobsFragment(c)
}

func (h *DashboardHandler) CreateUnstable(c echo.Context) error {
	if _, err := h.jobService.Enqueue(c.Request().Context(), "unstable-job"); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return h.JobsFragment(c)
}
