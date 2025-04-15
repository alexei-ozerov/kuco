package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"k8s.io/client-go/kubernetes"
)

type item string

func (i item) Title() string       { return string(i) }
func (i item) FilterValue() string { return string(i) }

type model struct {
	keys         *listKeyMap
	delegateKeys *delegateKeyMap

	containerHeight int
	containerWidth  int

	kubeContext  *kubernetes.Clientset
	currentView  int
	selectedItem string

	currentNamespace string
	currentPod       string
	currentContainer string
	currentLog       string

	displayList   list.Model
	namespaceList list.Model
	podList       list.Model
	containerList list.Model
	logList       list.Model

	execInput  textinput.Model
	execError  string
	execResult string
}

func newModel() model {
	var (
		delegateKeys = newDelegateKeyMap()
		listKeys     = newListKeyMap()
	)

	// Setup Kube Context
	ctx := InitKubeCtx()

	namespaceItemList := []list.Item{}
	namespaceList := GetNamespace(ctx)
	for _, listData := range namespaceList {
		namespaceItemList = append(namespaceItemList, item(listData))
	}

	// Setup list
	currentList := list.New(namespaceItemList, itemDelegate{}, 0, 0)
	currentList.Title = "Namespaces"
	currentList.Styles.Title = titleStyle
	currentList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.toggleTitleBar,
			listKeys.toggleStatusBar,
			listKeys.togglePagination,
			listKeys.toggleHelpMenu,
			listKeys.selection,
			listKeys.back,
		}
	}

	// Setup Exec Input TextInput Model
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return model{
		displayList:      currentList,
		keys:             listKeys,
		delegateKeys:     delegateKeys,
		kubeContext:      ctx,
		currentView:      0, // Namespace View
		currentContainer: "",
		currentPod:       "",
		currentLog:       "",
		currentNamespace: "",
		execInput:        ti,
		execError:        "",
		execResult:       "",
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		width := msg.Width - h
		height := msg.Height - v

		m.displayList.SetSize(width, height-9)
		m.containerWidth = width
		m.containerHeight = height
	case tea.KeyMsg:
		// Don't match any of the keys below if we're actively filtering.
		if m.displayList.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.keys.toggleTitleBar):
			v := !m.displayList.ShowTitle()
			m.displayList.SetShowTitle(v)
			m.displayList.SetShowFilter(v)
			m.displayList.SetFilteringEnabled(v)
			return m, nil

		case key.Matches(msg, m.keys.toggleStatusBar):
			m.displayList.SetShowStatusBar(!m.displayList.ShowStatusBar())
			return m, nil

		case key.Matches(msg, m.keys.togglePagination):
			m.displayList.SetShowPagination(!m.displayList.ShowPagination())
			return m, nil

		case key.Matches(msg, m.keys.toggleHelpMenu):
			m.displayList.SetShowHelp(!m.displayList.ShowHelp())
			return m, nil

		case key.Matches(msg, m.keys.back):
			if m.currentView > 0 {
				if m.currentView >= 4 {
					m.currentView = 2
					m.execInput.Reset()
					m.execResult = ""
					m.execError = ""
				}
				m.currentView -= 1
			}

			switch m.currentView {
			case 0:
				m.currentView = 0 // switch to pod view
				namespaceItemList := listToItemList(m.kubeContext, m.currentNamespace, m.currentView, m.currentPod, m.currentContainer)
				m.displayList = updateDisplayList(m, namespaceItemList)
			case 1:
				m.currentView = 1 // switch to pod view
				podItemList := listToItemList(m.kubeContext, m.currentNamespace, m.currentView, m.currentPod, m.currentContainer)
				m.displayList = updateDisplayList(m, podItemList)
			case 2:
				m.currentView = 2 // switch to pod view
				containerItemList := listToItemList(m.kubeContext, m.currentNamespace, m.currentView, m.currentPod, m.currentContainer)
				m.displayList = updateDisplayList(m, containerItemList)
			case 3:
				m.currentView = 3 // switch to pod view
				logItemList := listToItemList(m.kubeContext, m.currentNamespace, m.currentView, m.currentPod, m.currentContainer)
				m.displayList = updateDisplayList(m, logItemList)
			}

		case key.Matches(msg, m.keys.exec):
			if m.currentView == 2 {
				// Get selected container
				i, ok := m.displayList.SelectedItem().(item)
				if ok {
					m.currentContainer = string(i)
				}

				m.execResult = ""
				m.execError = ""
				m.execInput.Reset()
				m.execInput.SetValue("")
				m.currentView = 4

				return m, nil
			}

		case key.Matches(msg, m.keys.selection):
			i, ok := m.displayList.SelectedItem().(item)
			if ok {
				m.selectedItem = string(i)
			}

			switch m.currentView {
			case 0:
				m.currentNamespace = string(i)
				m.namespaceList = m.displayList
				m.currentView = 1 // switch to pod view
				podItemList := listToItemList(m.kubeContext, m.currentNamespace, m.currentView, m.currentPod, m.currentContainer)
				m.displayList = updateDisplayList(m, podItemList)
			case 1:
				m.currentPod = string(i)
				m.podList = m.displayList
				m.currentView = 2 // switch to container view
				containerItemList := listToItemList(m.kubeContext, m.currentNamespace, m.currentView, m.currentPod, m.currentContainer)
				m.displayList = updateDisplayList(m, containerItemList)
			case 2:
				m.currentContainer = string(i)
				m.containerList = m.displayList
				m.currentView = 3 // switch to log view
				logItemList := listToItemList(m.kubeContext, m.currentNamespace, m.currentView, m.currentPod, m.currentContainer)
				m.displayList = updateDisplayList(m, logItemList)
			case 3:
				m.currentLog = string(i)
			case 4:
				command := m.execInput.Value()

				containerName := m.currentContainer
				podName := m.currentPod
				namespace := m.currentNamespace

				output, stderr, err := ExecToPodThroughAPI(command, containerName, podName, namespace, nil)

				if len(stderr) != 0 {
					fmt.Println("STDERR:", stderr)
				}
				if err != nil {
					m.execError = fmt.Sprintf("Error occured while `exec`ing to the Pod %q, container %q, namespace %q, command %q. Error: %+v\n", podName, containerName, namespace, command, err)
				} else {
					m.execResult = output
				}

				var execResultItemList []list.Item
				execResultList := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
				for _, listData := range execResultList {
					execResultItemList = append(execResultItemList, item(listData))
				}

				m.displayList = updateDisplayList(m, execResultItemList)
				m.currentView = 5
			case 5:
				m.currentView = 2
				containerItemList := listToItemList(m.kubeContext, m.currentNamespace, m.currentView, m.currentPod, m.currentContainer)
				m.displayList = updateDisplayList(m, containerItemList)
			}

			return m, nil
		}

	}

	if m.currentView != 4 {
		// This will also call our delegate's update function.
		newListModel, cmd := m.displayList.Update(msg)
		m.displayList = newListModel
		cmds = append(cmds, cmd)
	} else {
		m.execInput, cmd = m.execInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	style := lipgloss.NewStyle().
		Width(m.containerWidth).
		Height(6).
		Padding(1).
		BorderStyle(lipgloss.RoundedBorder())

	var content string
	if m.currentView == 1 {
		content = ""
	} else if m.currentView == 3 {
		content = m.currentLog
	} else if m.currentView == 4 {
		content = m.execInput.View()
	}

	var textBlock string = style.Render(content)
	block := lipgloss.PlaceHorizontal(m.containerWidth, lipgloss.Center, textBlock)

	view := lipgloss.JoinVertical(lipgloss.Top, appStyle.Render("\n"+m.displayList.View()), block)

	return view
}

func main() {
	if _, err := tea.NewProgram(newModel(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
