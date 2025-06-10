package internal

import (
	"go/token"
)

// UserFunction rappresenta una funzione definita dall'utente
type UserFunction struct {
	Name       string
	InputTypes []string
	OutputType string
	FilePath   string
}

// ProcessorContext contiene lo stato del processore di codice
type ProcessorContext struct {
	Functions   map[string]*UserFunction
	FileSet     *token.FileSet
	CurrentFile string
	FuncFiles   []string
	TempDir     string
}

// Config contiene la configurazione dell'applicazione
type Config struct {
	Dir     string
	Verbose bool
	Help    bool
	Version bool
}
