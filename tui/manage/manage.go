package managetui

import (
	"fmt"
	"goful/core/model"
	create "goful/tui/requestcreate"
	list "goful/tui/requestlist"
	"log"
	"os"
	"os/exec"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"
)

type ActiveView int

const (
	List ActiveView = iota
	Create
	CreateComplex
	Update
)

var keys = []key.Binding{
	key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "Add simple"),
	),
	key.NewBinding(
		key.WithKeys("A"),
		key.WithHelp("A", "Add complex"),
	)}

type uiModel struct {
	list          list.Model
	create        create.Model
	createComplex create.Model
	active        ActiveView
	selected      list.Request
	width         int
	height        int
	debug         string
	help          help.Model
}

func (m uiModel) Init() tea.Cmd {
	return nil
}

func (m uiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			return m, tea.Quit
		case "a":
			if m.active != Create && m.active != CreateComplex {
				m.active = Create
				return m, nil
			}
		case "A":
			if m.active != Create && m.active != CreateComplex {
				m.active = CreateComplex
				return m, nil
			}
		}
	case list.RequestSelectedMsg:
		m.active = Update
	case create.CreateMsg:
		return m, tea.Quit
	}

	var cmd tea.Cmd
	switch m.active {
	case List:
		m.list, cmd = m.list.Update(msg)
	case Create:
		m.create, cmd = m.create.Update(msg)
	case CreateComplex:
		m.createComplex, cmd = m.createComplex.Update(msg)
	}
	return m, cmd
}

func (m uiModel) View() string {
	switch m.active {
	case List:
		return renderList(m)
	case Create:
		return renderCreate(m)
	case CreateComplex:
		return renderCreateComplex(m)
	default:
		return renderList(m)
	}
}

func renderList(m uiModel) string {
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		m.list.View())
}

func renderCreate(m uiModel) string {
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		m.create.View())
}

func renderCreateComplex(m uiModel) string {
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		m.createComplex.View())
}

func Start(loadedRequests []model.RequestMold) {
	var requests []list.Request

	for _, v := range loadedRequests {
		r := list.Request{
			Name:   v.Name(),
			Url:    v.Url(),
			Method: v.Method(),
		}
		requests = append(requests, r)
	}

	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()

	m := uiModel{list: list.New(requests, 0, 0, keys), create: create.New(false), createComplex: create.New(true), active: List}

	p := tea.NewProgram(m, tea.WithAltScreen())

	r, err := p.Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	if m, ok := r.(uiModel); ok {
		fileName := ""
		content := ""
		createFile := false
		if m.active == Create {
			fileName = fmt.Sprintf("%s.yaml", m.create.Name)
			createFile = true
			// TODO read from a template file
			content = fmt.Sprintf(`name: %s
prev_req: <call other request before this>
url: <your url>
method: <http method>
headers:
  <headers key-val list, e.g. X-Foo-Bar: SomeValue>
body: >
  <body, e.g. {
    <"id": 1,
    "name": "Jane">
  }>
`, m.create.Name)
		} else if m.active == CreateComplex {
			fileName = fmt.Sprintf("%s.star", m.createComplex.Name)
			// TODO read from template
			content = fmt.Sprintf(`"""
meta:name: %s
meta:prev_req: <call other request before this>
doc:url: <your url for display>
doc:method: <your http method for display>
"""
# insert contents of your script here, for more see https://github.com/google/starlark-go/blob/master/doc/spec.md
# Request url
url = ""
# HTTP method
method = ""
# HTTP headers, e.g. { "X-Foo": "bar", "X-Foos": [ "Bar1", "Bar2" ] }
headers = {}
# Request body, e.g. { "id": 1, "people": [ {"name": "Joe"}, {"name": "Jane"}, ] }
body = {}
`, m.createComplex.Name)
			createFile = true
		}

		if !createFile {
			return
		}

		log.Printf("About to create new request with name %v", fileName)
		if len(fileName) > 0 {
			file, err := os.Create("tmp/" + fileName)
			if err == nil {
				defer file.Close()
				// TODO handle err
				file.WriteString(content)
				file.Sync()
				filename := file.Name()
				editor := viper.GetString("editor")
				if editor == "" {
					log.Fatal("Editor is not configured through configuration file or $EDITOR environment variable.")
				}

				cmd := exec.Command(editor, filename)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr

				err = cmd.Run()
				if err != nil {
					log.Printf("Failed to open file with editor: %v", err)
				}
				log.Printf("Successfully edited file %v", file.Name())
				fmt.Printf("Saved new request to file %v", file.Name())
			} else {
				log.Printf("Failed to create file %v", err)
			}
		}
	}
}
