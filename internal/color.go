package internal

import "fmt"

func identString(s string) string  { return s }
func redString(s string) string    { return fmt.Sprintf("\x1b[31m%s\x1b[0m", s) }
func greenString(s string) string  { return fmt.Sprintf("\x1b[32m%s\x1b[0m", s) }
func yellowString(s string) string { return fmt.Sprintf("\x1b[33m%s\x1b[0m", s) }
func cyanString(s string) string   { return fmt.Sprintf("\x1b[36m%s\x1b[0m", s) }
