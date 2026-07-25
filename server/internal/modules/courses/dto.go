package courses

type CreateCourseRequest struct {
	Title         string `json:"title" binding:"required,min=2,max=200"`
	Summary       string `json:"summary" binding:"max=2000"`
	DescriptionMD string `json:"description_md"`
}

type UpdateCourseRequest struct {
	Title         string `json:"title" binding:"required,min=2,max=200"`
	Summary       string `json:"summary" binding:"max=2000"`
	DescriptionMD string `json:"description_md"`
}

type ModuleRequest struct {
	Title    string `json:"title" binding:"required,min=1,max=200"`
	Position int    `json:"position"`
}

type LessonRequest struct {
	Title     string `json:"title" binding:"required,min=1,max=200"`
	ContentMD string `json:"content_md"`
	Position  int    `json:"position"`
}
