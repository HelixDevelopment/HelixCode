// Template Library Example

package main

import (
	"fmt"

	"dev.helix.code/internal/template"
)

func main() {
	fmt.Println("=== Template Library Example ===\n")

	mgr := template.NewManager()
	mgr.RegisterBuiltinTemplates()

	// Create custom API handler template
	apiHandler := template.NewTemplate("API Handler", "REST API handler", template.TypeCode)
	apiHandler.SetContent(`func Handle{{name}}(w http.ResponseWriter, r *http.Request) {
	// {{description}}

	{{implementation}}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}`)

	apiHandler.AddVariable(template.Variable{Name: "name", Required: true})
	apiHandler.AddVariable(template.Variable{Name: "description", Required: false, DefaultValue: "Handler implementation"})
	apiHandler.AddVariable(template.Variable{Name: "implementation", Required: true})
	apiHandler.AddTag("http")
	apiHandler.AddTag("api")

	mgr.Register(apiHandler)

	// Render the template
	result, _ := mgr.Render(apiHandler.ID, map[string]interface{}{
		"name":           "Users",
		"description":    "Handles user operations",
		"implementation": "users := getUsersFromDB()\nresponse := map[string]interface{}{\"users\": users}",
	})

	fmt.Println("Generated API Handler:")
	fmt.Println(result)
	fmt.Println()

	// Use built-in templates
	funcCode, _ := mgr.RenderByName("Function", map[string]interface{}{
		"function_name": "ValidateEmail",
		"parameters":    "email string",
		"return_type":   "bool",
		"body":          "re := regexp.MustCompile(`^[a-z0-9._%+\\-]+@[a-z0-9.\\-]+\\.[a-z]{2,}$`)\nreturn re.MatchString(email)",
	})

	fmt.Println("Generated Function:")
	fmt.Println(funcCode)

	fmt.Printf("\nTotal templates: %d\n", mgr.Count())
}
