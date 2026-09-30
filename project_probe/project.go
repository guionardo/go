package projectprobe

type (
	// ProjectData describes a probed project folder.
	ProjectData struct {
		// Folder is the cleaned folder path as given —
		// filepath.Clean(as-given); relative input stays relative (D-08).
		Folder string

		// Language is the detected project language.
		Language Language

		// Name is the project name reported by the winning detector.
		Name string

		// Version is the project version reported by the winning detector.
		Version string

		// Description is the project description reported by the winning
		// detector.
		Description string
	}

	// Language identifies the primary programming language of a project.
	Language string
)

const (
	// LanguageGo identifies a Go project.
	LanguageGo Language = "Go"

	// LanguagePython identifies a Python project.
	LanguagePython Language = "Python"

	// LanguageJavaScript identifies a JavaScript project.
	LanguageJavaScript Language = "JavaScript"

	// LanguageCSharp identifies a C#/.NET project.
	LanguageCSharp Language = "C#/.NET"

	// LanguageRust identifies a Rust project.
	LanguageRust Language = "Rust"

	// LanguageJava identifies a Java project.
	LanguageJava Language = "Java"

	// LanguagePHP identifies a PHP project.
	LanguagePHP Language = "PHP"

	// LanguageUnknown identifies an unrecognized project. It is an explicit
	// sentinel value, never the zero value "" (D-07).
	LanguageUnknown Language = "unknown"
)
