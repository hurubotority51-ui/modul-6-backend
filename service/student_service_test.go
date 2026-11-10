package service

import (
	"fmt"
	"testing"
	"time"

	"modul6/helper"
	"modul6/model"
	"modul6/repository"
)

func TestListCursorDoesNotRepeatRecordsAfterNewInsert(t *testing.T) {
	repo := repository.NewInMemoryStudentRepository()
	base := time.Date(2026, 9, 13, 14, 0, 0, 0, time.UTC)
	for i := 1; i <= 4; i++ {
		repo.Create(model.Student{
			NIM: fmt.Sprintf("%08d", i), Name: fmt.Sprintf("Student %d", i),
			CreatedAt: base,
		})
	}
	svc := NewStudentService(repo)

	firstPage, firstMeta := svc.List(model.CursorQuery{Limit: 2})
	if len(firstPage) != 2 || !firstMeta.HasMore || firstMeta.NextCursor == "" {
		t.Fatalf("first page/meta = %#v / %#v", firstPage, firstMeta)
	}
	cursor, err := helper.DecodeCursor(firstMeta.NextCursor)
	if err != nil {
		t.Fatalf("invalid next cursor: %v", err)
	}

	repo.Create(model.Student{
		NIM: "00000099", Name: "New top record", CreatedAt: base,
	})
	secondPage, secondMeta := svc.List(model.CursorQuery{Limit: 2, After: &cursor})
	if len(secondPage) != 2 || secondMeta.HasMore || secondMeta.NextCursor != "" {
		t.Fatalf("second page/meta = %#v / %#v", secondPage, secondMeta)
	}
	if secondPage[0].ID == firstPage[0].ID || secondPage[0].ID == firstPage[1].ID ||
		secondPage[1].ID == firstPage[0].ID || secondPage[1].ID == firstPage[1].ID {
		t.Fatalf("cursor page repeated an earlier row: first=%#v second=%#v", firstPage, secondPage)
	}
	if secondPage[0].Name == "New top record" || secondPage[1].Name == "New top record" {
		t.Fatal("newly inserted top row should not shift the cursor page")
	}
}
