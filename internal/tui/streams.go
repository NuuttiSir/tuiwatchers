package tui

import (
	"fmt"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/NuuttiSir/tuiwatchers/internal/storage"
)

type StreamsModel struct {
	State           page
	Err             error
	Channels        []ChannelInfo
	ChannelList     list.Model
	SelectedChannel string
	BroadcasterIDs  map[string]string
	TokenFile       storage.TokenFile
	WindowWidth     int
	WindowHeight    int
}
type ChannelInfo struct {
	BroadcasterName string
	Login           string
	GameName        string
	ViewCount       int
}

func (chInfo ChannelInfo) FilterValue() string { return chInfo.BroadcasterName + " " + chInfo.GameName }
func (chInfo ChannelInfo) Title() string       { return chInfo.BroadcasterName }
func (chInfo ChannelInfo) Description() string {
	return fmt.Sprintf("%s - %d viewers", chInfo.GameName, chInfo.ViewCount)
}

func InitialStreamsModel(channels []ChannelInfo, ids map[string]string, tokenFile storage.TokenFile, width, height int) StreamsModel {
	channelList := newChannelList(channels)
	channelList.SetSize(width, height)
	return StreamsModel{
		State:          PageStreams,
		Channels:       channels,
		ChannelList:    channelList,
		BroadcasterIDs: ids,
		TokenFile:      tokenFile,
	}
}
func newChannelList(channels []ChannelInfo) list.Model {
	items := make([]list.Item, 0, len(channels))
	for _, channel := range channels {
		items = append(items, channel)
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(1)

	channelList := list.New(items, delegate, 0, 0)
	channelList.Title = "Live channels"
	channelList.SetFilteringEnabled(false)
	channelList.SetShowStatusBar(false)
	channelList.SetShowHelp(false)
	channelList.DisableQuitKeybindings()
	return channelList
}

func (sm StreamsModel) Init() tea.Cmd {
	return nil
}

func (sm StreamsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		sm.WindowWidth = msg.Width
		sm.WindowHeight = msg.Height
		h, v := docStyle.GetFrameSize()
		sm.ChannelList.SetSize(sm.WindowWidth-h, sm.WindowHeight-v)
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			sm.State = PageQuitting
			return sm, tea.Quit
		case "enter":
			item, ok := sm.ChannelList.SelectedItem().(ChannelInfo)
			if ok {
				sm.SelectedChannel = item.Login
				return sm, tea.Quit
			}
			return sm, nil
		}
	}
	var cmd tea.Cmd
	sm.ChannelList, cmd = sm.ChannelList.Update(msg)
	return sm, cmd
}

func (sm StreamsModel) View() tea.View {
	str := docStyle.Render(sm.ChannelList.View())
	v := tea.NewView(str)
	v.AltScreen = true
	return v
}
