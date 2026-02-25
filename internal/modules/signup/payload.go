package signup

type IndividualPayload struct {
	Matricula string `json:"matricula"`
	Nome      string `json:"nome"`
	Semestre  int    `json:"semestre"`
}

type TeamPayload struct {
	Participants []struct {
		Matricula string `json:"matricula"`
		Nome      string `json:"nome"`
		Semestre  int    `json:"semestre"`
	} `json:"participants"`
	IsDraw bool `json:"isDraw"`
}
