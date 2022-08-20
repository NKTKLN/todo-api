package models

type ApiShowLists struct {
	Lists []ListsData `json:"lists"`
}

type ApiListData struct {
	Name    string `json:"name" example:"List of products"`
	Comment string `json:"comment" example:"Products needed for the party"`
}

type ListsData struct {
	Id      int    `json:"id" example:"1023456789"`
	Name    string `json:"name" example:"List of products"`
	Comment string `json:"comment" example:"Products needed for the party"`
	Index   int    `json:"index" example:"0"`
}

type ListEditData struct {
	Id      int    `json:"id" example:"1023456789"`
	Name    string `json:"name" example:"New list of products"`
	Comment string `json:"comment" example:"Products needed for the party"`
	Index   int    `json:"index" example:"1"`
}
