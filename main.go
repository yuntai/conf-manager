package main

func main() {
	logEntry := configureLogger("main")

	m, err := NewConfMaster(&MasterConfig{})

	if err != nil {
		logEntry.Errorf("Failed to create ConfMaster err: %v\n", err)
		return
	}

	m.Run()
}
