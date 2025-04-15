package main

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"k8s.io/client-go/kubernetes"
)

func updateDisplayList(m model, itemList []list.Item) list.Model {
	listKeys := m.keys
	currentList := list.New(itemList, itemDelegate{}, 0, 0)

	var title string
	switch m.currentView {
	case 0:
		title = "Namespaces"
	case 1:
		title = "Pods"
	case 2:
		title = "Containers"
	case 3:
		title = "Logs"
	}

	currentList.Title = title
	currentList.Styles.Title = titleStyle
	currentList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.toggleTitleBar,
			listKeys.toggleStatusBar,
			listKeys.togglePagination,
			listKeys.toggleHelpMenu,
			listKeys.selection,
		}
	}

	currentList.SetSize(m.containerWidth, m.containerHeight)

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
