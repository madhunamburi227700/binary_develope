package repository

// ProjectRepository handles project-related database operations
type ProjectRepository struct {
	*BaseRepository
}

// NewProjectRepository creates a new project repository
func NewProjectRepository() *ProjectRepository {
	return &ProjectRepository{
		BaseRepository: NewBaseRepository("projects"),
	}
}
