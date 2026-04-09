# VHS tape file for azboard demo
# Run with: vhs demo.tape

Output demo.gif

Set Shell bash
Set FontSize 12
Set Width 1800
Set Height 900
Set Padding 10
Set TypingSpeed 40ms

# Build and launch in demo mode
Hide
Type "go build -o azboard . && ./azboard --demo"
Enter
Sleep 2s
Show

# --- PR List ---
Sleep 3s

# Navigate down through PRs
Sleep 500ms
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 1s

# Go back to top
Type "g"
Sleep 1s

# Filter PRs
Type "/"
Sleep 500ms
Type "batch"
Sleep 2s

# Clear filter and open PR
Escape
Sleep 1s
Enter
Sleep 3s

# --- PR Detail (overview with threads) ---
Sleep 3s

# Scroll down to see comments
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 2s

# Navigate to next thread
Type "n"
Sleep 2s

# Navigate to next thread (inline comment)
Type "n"
Sleep 2s

# Switch to files pane
Type "f"
Sleep 3s

# Open a directory
Enter
Sleep 1s
Type "j"
Sleep 500ms

# View diff
Enter
Sleep 3s

# Scroll through diff
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 2s

# Go back to PR list
Escape
Sleep 500ms
Escape
Sleep 500ms
Escape
Sleep 1s

# --- Switch to Work Items tab ---
Tab
Sleep 3s

# Navigate work items
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 1s

# Go back up
Type "k"
Sleep 400ms
Type "k"
Sleep 400ms
Type "k"
Sleep 1s

# Open first work item
Enter
Sleep 3s

# Scroll through work item detail
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 2s

# Go back to list
Escape
Sleep 2s

# Final pause on work items list
Sleep 2s
