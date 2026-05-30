# Proposed Changes

## 1. Typo

| File | Line | Current | Fix |
|------|------|---------|-----|
| `authPageUi.go` | 68 | `"Successfull authentication"` | `"Successful authentication"` |

---

## 2. Cyclomatic complexity — `(AuthModel).Update()` (authPageUi.go:60)

The `AuthUserTokenMessage` case (lines 79–136) does too much: validates token, fetches user, saves token, loads it back, fetches followed channels, filters live channels, builds channel lists, and can fail in ~6 ways. This drives the complexity score to 19.

**Fix:** Extract the body of `case AuthUserTokenMessage:` into a separate method:

```go
func (am AuthModel) handleAuthUserToken(msg AuthUserTokenMessage) (AuthModel, tea.Cmd) {
    if msg.Err != nil || msg.UserToken.AccessToken == "" {
        am.State = pageQuitting
        am.Err = msg.Err
        return am, tea.Quit
    }

    authUser := getAuthenticatedUser(clientID, msg.UserToken)
    if authUser.ID == "" {
        am.State = pageQuitting
        am.Err = errors.New("could not fetch user data")
        return am, tea.Quit
    }

    if err := saveToken(tokenFilePath, msg.UserToken.AccessToken, authUser.ID); err != nil {
        am.State = pageQuitting
        am.Err = err
        return am, tea.Quit
    }
    tokenFile, err := tokenLoad(tokenFilePath)
    if err != nil {
        am.State = pageQuitting
        am.Err = err
        return am, tea.Quit
    }
    followDataList := getFollowedChannels(tokenFile.UserID, clientID, AccessToken{AccessToken: tokenFile.AccessToken})
    if len(followDataList.Data) == 0 {
        am.State = pageQuitting
        am.Err = errors.New("no followed channels found")
        return am, tea.Quit
    }
    channels := make([]ChannelInfo, 0, len(followDataList.Data))
    ids := make(map[string]string)
    for _, channel := range followDataList.Data {
        if channel.Type != "live" {
            continue
        }
        channels = append(channels, ChannelInfo{
            BroadcasterName: channel.UserName,
            GameName:        channel.GameName,
            ViewCount:       channel.ViewerCount,
        })
        ids[channel.UserName] = channel.UserID
    }

    if len(channels) == 0 {
        am.State = pageQuitting
        am.Err = errors.New("no live channels found")
        return am, tea.Quit
    }

    return am, func() tea.Msg {
        return AuthSuccessMessage{
            ChannelList:    channels,
            BroadcasterIDs: ids,
            TokenFile:      tokenFile,
        }
    }
}
```

Then replace the case body with:

```go
case AuthUserTokenMessage:
    return am.handleAuthUserToken(msg)
```

---

## 3. Bug: space vs empty string (auth.go:134)

```go
// Current
if deviceCode.DeviceCode == " " {

// Fix
if deviceCode.DeviceCode == "" {
```

---

## 4. Swallowed errors (api.go)

`getFollowedChannels` and `getAuthenticatedUser` print errors to stdout and return zero-value structs. The caller cannot distinguish success from failure.

**Fix:** Return `error` from these functions:

```go
func getFollowedChannels(userID, clientID string, userToken AccessToken) (FollowDataList, error) {
    req, err := http.NewRequest("GET", TwitchAPIURL+"streams/followed", nil)
    if err != nil {
        return FollowDataList{}, fmt.Errorf("create request: %w", err)
    }
    // ... same setup ...
    resp, err := client.Do(req)
    if err != nil {
        return FollowDataList{}, fmt.Errorf("do request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return FollowDataList{}, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
    }

    var followDataList FollowDataList
    if err := json.NewDecoder(resp.Body).Decode(&followDataList); err != nil {
        return FollowDataList{}, fmt.Errorf("decode: %w", err)
    }
    return followDataList, nil
}
```

Same pattern for `getAuthenticatedUser`. Then update callers to handle the error.

---

## 5. Lowercase error messages

Go convention: error strings start with lowercase unless they contain proper nouns.

| File | Line | Current | Fix |
|------|------|---------|-----|
| `authPageUi.go` | 89 | `"Could not fetch user data"` | `"could not fetch user data"` |
| `auth.go` | 135 | `"Could not get device code"` | `"could not get device code"` |

---

## 6. Variable shadowing (authPageUi.go:159, 166)

```go
var str string                              // line 159

// inside switch/case:
str := fmt.Sprintf("%s %s", ...)            // line 166 — shadows outer str
```

**Fix:** Use `str =` (assignment) instead of `str :=` (short declaration):

```go
str = fmt.Sprintf("%s %s", am.Spinner.View(), status)
```

Or remove the outer `var str string` and declare where first needed.

---

## 7. Unused blank import (main.go:10)

```go
_ "charm.land/bubbles/v2/list"
```

Likely leftover from refactoring. Remove if the build passes without it.

---

## 8. Commented-out dead code (main.go:172–182)

Delete lines 172–182 (the commented-out block at the bottom of `main()`).

---

## 9. File permission (storage.go:19)

```go
// Current
return os.WriteFile(path, bytesWrite, 0666)

// Fix
return os.WriteFile(path, bytesWrite, 0o644)
```

`0666` is world-writable. `0644` is the conventional file permission.

---

## 10. JSON indent (storage.go:15)

```go
// Current
json.MarshalIndent(file, "", " ")

// Fix
json.MarshalIndent(file, "", "\t")
```

Go convention is tabs, not spaces.

---

## 11. Inconsistent naming (emotes.go)

Two exported functions for emote images with unclear distinction:

- `EmoteImage(id string) ([]byte, bool)` — synchronous cache lookup
- `GetEmoteImage(id string) tea.Cmd` — async HTTP fetch

**Fix:** Rename to signal the difference:

| Current | Fix |
|---------|-----|
| `EmoteImage` | `emoteImage` (unexport — internal helper) |
| `GetEmoteImage` | `FetchEmoteImage` (signals network I/O) |

---

## 12. Repeated View pattern (authPageUi.go:158–187)

The `tea.NewView(docStyle.Render(str))` + `.AltScreen = true` pattern repeats 4 times.

**Fix (optional):** Extract a helper:

```go
func renderView(str string) tea.View {
    v := tea.NewView(docStyle.Render(str))
    v.AltScreen = true
    return v
}
```
