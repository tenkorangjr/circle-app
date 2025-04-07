package requestmodel

type CommentRequest struct {
	Content string `validate:"required,max=100"`
}
