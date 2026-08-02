package batches

type CreateBatchRequest struct {
	CourseID int    `json:"course_id" binding:"required"`
	Name     string `json:"name" binding:"required,min=1,max=200"`
}

type UpdateBatchRequest struct {
	Name string `json:"name" binding:"required,min=1,max=200"`
}

type AssignTutorRequest struct {
	TutorID         int    `json:"tutor_id" binding:"required"`
	StartDate       string `json:"start_date" binding:"required"`
	ExpectedEndDate string `json:"expected_end_date" binding:"required"`
}

type CreateFolderRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=255"`
	ParentID *int   `json:"parent_id"`
}

type RenameNodeRequest struct {
	Name string `json:"name" binding:"required,min=1,max=255"`
}

type EnrollStudentsRequest struct {
	StudentIDs []int `json:"student_ids" binding:"required,min=1"`
}
