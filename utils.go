package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"k8s.io/client-go/kubernetes"
)

func updateDisplayList(m model, itemList []list.Item) list.Model {
	listKeys := m.keys
	currentList := list.New(itemList, itemDelegate{}, 0, 0)

	if m.currentView != 4 {
		currentList.AdditionalShortHelpKeys = func() []key.Binding {
			return []key.Binding{
				listKeys.selection,
				listKeys.back,
			}
		}
	}

	var title string
	switch m.currentView {
	case 0:
		title = "[KUCO] Namespaces"
		currentList.AdditionalShortHelpKeys = func() []key.Binding {
			return []key.Binding{
				listKeys.selection,
			}
		}
	case 1:
		title = "[KUCO] Pods"
	case 2:
		title = "[KUCO] Containers"
		currentList.AdditionalShortHelpKeys = func() []key.Binding {
			return []key.Binding{
				listKeys.selection,
				listKeys.back,
				listKeys.exec,
			}
		}
	case 3:
		title = "[KUCO] Logs"
		currentList.Help.ShowAll = false
	case 4:
		title = fmt.Sprintf("[KUCO] Command Output\n> %s", m.execInput.Value())

		currentList.AdditionalShortHelpKeys = func() []key.Binding {
			return []key.Binding{
				listKeys.selection,
				listKeys.back,
				listKeys.exec,
			}
		}
	}

	currentList.Title = title
	currentList.Styles.Title = titleStyle
	currentList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.toggleTitleBar,
			listKeys.toggleStatusBar,
			listKeys.togglePagination,
			listKeys.toggleHelpMenu,
		}
	}

	cHeight := m.containerHeight - 9
	currentList.SetSize(m.containerWidth, cHeight)

	return currentList
}

func listToItemList(kubeContext *kubernetes.Clientset, namespace string, viewMode int, podName string, containerName string) []list.Item {
	var stringList []string
	switch viewMode {
	case 0:
		stringList = GetNamespace(kubeContext)
	case 1:
		stringList = GetPods(kubeContext, namespace)
	case 2:
		stringList = GetContainers(kubeContext, namespace, podName)
	case 3:
		stringList = GetLogs(kubeContext, namespace, podName, containerName)
	}

	itemList := []list.Item{}
	for _, listData := range stringList {
		itemList = append(itemList, item(listData))
	}

	return itemList
}
