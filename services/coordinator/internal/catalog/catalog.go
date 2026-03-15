package catalog

type RateCard struct {
	MinJobUSDC   string `json:"min_job_usdc"`
	Input1MUSDC  string `json:"input_1m_usdc"`
	Output1MUSDC string `json:"output_1m_usdc"`
}

type Entry struct {
	ModelID  string   `json:"model_id"`
	MemoryGB int      `json:"memory_gb"`
	Backend  string   `json:"backend"`
	Tier     string   `json:"tier"`
	RateCard RateCard `json:"rate_card"`
}

func DefaultEntries() []Entry {
	return []Entry{
		{
			ModelID:  "qwen3.5-4b-mlx-4bit",
			MemoryGB: 16,
			Backend:  "mlx",
			Tier:     "local-smoke",
			RateCard: RateCard{
				MinJobUSDC:   "0.01",
				Input1MUSDC:  "10.00",
				Output1MUSDC: "20.00",
			},
		},
		{
			ModelID:  "qwen3.5-9b",
			MemoryGB: 16,
			Backend:  "mlx",
			Tier:     "smoke",
			RateCard: RateCard{
				MinJobUSDC:   "0.01",
				Input1MUSDC:  "20.00",
				Output1MUSDC: "30.00",
			},
		},
		{
			ModelID:  "qwen3.5-35b-a3b",
			MemoryGB: 64,
			Backend:  "mlx",
			Tier:     "standard",
			RateCard: RateCard{
				MinJobUSDC:   "0.05",
				Input1MUSDC:  "80.00",
				Output1MUSDC: "100.00",
			},
		},
		{
			ModelID:  "qwen3.5-122b-a10b",
			MemoryGB: 128,
			Backend:  "mlx",
			Tier:     "premium",
			RateCard: RateCard{
				MinJobUSDC:   "0.25",
				Input1MUSDC:  "400.00",
				Output1MUSDC: "550.00",
			},
		},
	}
}
