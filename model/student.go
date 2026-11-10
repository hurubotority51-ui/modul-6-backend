package model

import "time"

type Student struct {
	ID        int       `json:"id"`
	NIM       string    `json:"nim"`
	Name      string    `json:"name"`
	Grade     float64   `json:"grade"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateStudentRequest struct {
	NIM      string  `json:"nim" validate:"required,min=3,max=20,nim"`
	Name     string  `json:"name" validate:"required,min=2,max=100"`
	Grade    float64 `json:"grade" validate:"gte=0,lte=4"`
	IsActive bool    `json:"is_active"`
}

type PatchStudentRequest struct {
	NIM      *string  `json:"nim" validate:"omitnil,min=3,max=20,nim"`
	Name     *string  `json:"name" validate:"omitnil,min=2,max=100"`
	Grade    *float64 `json:"grade" validate:"omitnil,gte=0,lte=4"`
	IsActive *bool    `json:"is_active" validate:"omitnil"`
}

func (r PatchStudentRequest) IsEmpty() bool {
	return r.NIM == nil && r.Name == nil && r.Grade == nil && r.IsActive == nil
}

type Cursor struct {
	CreatedAt time.Time
	ID        int
}

type CursorQuery struct {
	Limit    int
	Search   string
	IsActive *bool
	After    *Cursor
}

type CursorMeta struct {
	Limit      int    `json:"limit"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}
