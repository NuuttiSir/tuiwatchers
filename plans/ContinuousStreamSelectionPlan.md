# Continuous Stream Selection Plan (Step-by-Step)

This plan describes how to keep the TuiWatchers UI open after a stream is selected, allowing the user to seamlessly switch between streams. When a new stream is selected, the application will kill the currently running MPV and Chat processes and start the new ones.

## 1) Track Active Processes in the Application State

**Goal:** Allow the UI to keep track of external processes so they can be managed (killed).

**Steps:**
- Open `streamsPageUi.go`.
- Modify the `StreamsModel` struct to store references to the active MPV and Chat commands.

```go
import "os/exec"

type StreamsModel struct {
	// ... existing fields ...
	ActiveMPV  *exec.Cmd
	ActiveChat *exec.Cmd
}
```

## 2) Prevent the App from Quitting on Stream Selection

**Goal:** Stop the main program loop from exiting when a user selects a stream.

**Steps:**
- In `streamsPageUi.go`, locate the `Update` method for `StreamsModel`.
- Under `case "enter":`, remove the lines that set the state to quitting and exit the program:
  ```go
  // REMOVE THESE:
  // sm.State = pageQuitting
  // return sm, tea.Quit
  ```

## 3) Modify Spawner Functions to Run Asynchronously

**Goal:** Start the processes without blocking the UI, and return the process handle so we can kill it later.

**Steps:**
- In `player.go`, update `startMPVWithStream` to accept the channel name (instead of the whole UI model) and use `Start()` instead of `CombinedOutput()`:
  ```go
  func startMPVWithStream(channelName string) (*exec.Cmd, error) {
      // NOTE: You might want to remove the standard output waiting so it doesn't block
      mpvInstance := exec.Command("/usr/bin/mpv", "https://twitch.tv/"+channelName)
      err := mpvInstance.Start()
      return mpvInstance, err
  }
  ```
- In `chat.go`, update `spawnChatWindow` to return the `*exec.Cmd` and error:
  ```go
  func spawnChatWindow(broadcasterID, userID, accessToken string) (*exec.Cmd, error) {
      cmd := exec.Command("/usr/bin/ghostty", "-e", "bash", "-c", "./tuiwatchers --chat "+clientID+" "+broadcasterID+" "+userID+" "+accessToken+";exec bash")
      err := cmd.Start()
      return cmd, err
  }
  ```

## 4) Implement the Kill-and-Swap Logic

**Goal:** When the user selects a new stream, terminate any existing streams before starting the new one.

**Steps:**
- Back in `streamsPageUi.go`, inside the `Update` method under `case "enter":`, implement the swap logic:

```go
case "enter":
    item, ok := sm.ChannelList.SelectedItem().(ChannelInfo)
    if ok {
        sm.SelectedChannel = item.BroadcasterName
        broadcasterID := sm.BroadcasterIDs[item.BroadcasterName]

        // 1. Kill existing MPV process
        if sm.ActiveMPV != nil && sm.ActiveMPV.Process != nil {
            sm.ActiveMPV.Process.Kill()
        }

        // 2. Kill existing Chat process
        if sm.ActiveChat != nil && sm.ActiveChat.Process != nil {
            sm.ActiveChat.Process.Kill()
        }

        // 3. Start the new MPV process
        mpvCmd, _ := startMPVWithStream(item.BroadcasterName)
        sm.ActiveMPV = mpvCmd

        // 4. Start the new Chat process
        chatCmd, _ := spawnChatWindow(broadcasterID, sm.TokenFile.UserID, sm.TokenFile.AccessToken)
        sm.ActiveChat = chatCmd
    }
    // Return without quitting!
    return sm, nil
```

## 5) Cleanup on Application Exit

**Goal:** Ensure we don't leave zombie MPV or ghostty windows open when the user fully quits TuiWatchers.

**Steps:**
- In `streamsPageUi.go`, inside the `Update` method under `case "q", "esc", "ctrl+c":`:
- Add the same kill logic before returning `tea.Quit`:

```go
case "q", "esc", "ctrl+c":
    if sm.ActiveMPV != nil && sm.ActiveMPV.Process != nil {
        sm.ActiveMPV.Process.Kill()
    }
    if sm.ActiveChat != nil && sm.ActiveChat.Process != nil {
        sm.ActiveChat.Process.Kill()
    }
    sm.State = pageQuitting
    return sm, tea.Quit
```

## 6) Disconnect Spawning from main.go

**Goal:** `main.go` used to spawn the stream because it caught the result of the `program.Run()`. Now the UI handles it internally.

**Steps:**
- In `main.go`, near the end of the file, remove the calls to spawn processes, since the UI is doing it now.
  ```go
  // REMOVE THESE from main.go:
  // spawnChatWindow(...)
  // startMPVWithStream(...)
  ```
