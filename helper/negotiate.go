package helper

import (
	"encoding/csv"
	"strconv"
	"strings"

	"modul6/model"

	"github.com/gofiber/fiber/v2"
)

const (
	FormatJSON = fiber.MIMEApplicationJSON
	FormatCSV  = "text/csv"
)

func Negotiate(c *fiber.Ctx, offered ...string) (string, error) {
	accept := strings.TrimSpace(c.Get(fiber.HeaderAccept))
	if accept == "" || accept == "*/*" {
		return offered[0], nil
	}
	chosen := c.Accepts(offered...)
	if chosen == "" {
		return "", NotAcceptable(
			"format yang diminta tidak tersedia, pilih salah satu dari: " +
				strings.Join(offered, ", "))
	}
	return chosen, nil
}

func WriteStudentsCSV(c *fiber.Ctx, students []model.Student) error {
	c.Set(fiber.HeaderContentType, FormatCSV+"; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="students.csv"`)
	var buffer strings.Builder
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"id", "nim", "name", "grade", "is_active", "created_at"}); err != nil {
		return Internal(err)
	}
	for _, student := range students {
		if err := writer.Write([]string{
			strconv.Itoa(student.ID), student.NIM, student.Name,
			strconv.FormatFloat(student.Grade, 'f', -1, 64),
			strconv.FormatBool(student.IsActive),
			student.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}); err != nil {
			return Internal(err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return Internal(err)
	}
	return c.SendString(buffer.String())
}
