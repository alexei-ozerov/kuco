package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
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
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		width := msg.Width - h
		height := msg.Height - v

		m.displayList.SetSize(width, height)
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
			}

			return m, nil
		}

	}

	// This will also call our delegate's update function.
	newListModel, cmd := m.displayList.Update(msg)
	m.displayList = newListModel
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return appStyle.Render("\n" + m.displayList.View())
}

func main() {
	if _, err := tea.NewProgram(newModel(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
