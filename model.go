package main

type UserRequest struct {
	LongURL string `json:"long_url" binding:"required,url"`
}
