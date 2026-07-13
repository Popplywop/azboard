using ConsoleForge.Core;

namespace devo.ui;

/// <summary>Central key binding table — port of internal/ui/keys.go.
/// Patterns drive KeyMap dispatch; HelpKey/HelpDescription drive the help bar.</summary>
internal static class KeyBindings
{
    public static readonly KeyBinding Quit = new([KeyPattern.Plain(ConsoleKey.Q), KeyPattern.WithCtrl(ConsoleKey.C)], "q", "quit");
    public static readonly KeyBinding Back = new([KeyPattern.Plain(ConsoleKey.Escape)], "esc", "back");
    public static readonly KeyBinding Select = new([KeyPattern.Plain(ConsoleKey.Enter)], "enter", "select");
    public static readonly KeyBinding Refresh = new([KeyPattern.Plain(ConsoleKey.R)], "r", "refresh");
    public static readonly KeyBinding Help = new([KeyPattern.WithShift(ConsoleKey.Oem2)], "?", "help");
    public static readonly KeyBinding Tab = new([KeyPattern.Plain(ConsoleKey.Tab)], "tab", "switch tab");
    public static readonly KeyBinding Up = new([KeyPattern.Plain(ConsoleKey.UpArrow), KeyPattern.Plain(ConsoleKey.K)], "↑/k", "up");
    public static readonly KeyBinding Down = new([KeyPattern.Plain(ConsoleKey.DownArrow), KeyPattern.Plain(ConsoleKey.J)], "↓/j", "down");
    public static readonly KeyBinding Filter = new([KeyPattern.Plain(ConsoleKey.Oem2)], "/", "filter");
    public static readonly KeyBinding NextThread = new([KeyPattern.Plain(ConsoleKey.N)], "n", "next thread");
    public static readonly KeyBinding PrevThread = new([KeyPattern.WithShift(ConsoleKey.N)], "N", "prev thread");
    public static readonly KeyBinding Reply = new([KeyPattern.Plain(ConsoleKey.C)], "c", "reply to thread");
    public static readonly KeyBinding NewThread = new([KeyPattern.WithShift(ConsoleKey.C)], "C", "new comment");
    public static readonly KeyBinding ResolveThread = new([KeyPattern.Plain(ConsoleKey.S)], "s", "resolve/reactivate");
    public static readonly KeyBinding Approve = new([KeyPattern.Plain(ConsoleKey.A)], "a", "approve");
    public static readonly KeyBinding ApproveWithSuggestions = new([KeyPattern.WithShift(ConsoleKey.A)], "A", "approve w/ suggestions");
    public static readonly KeyBinding Reject = new([KeyPattern.Plain(ConsoleKey.X)], "x", "reject");
    public static readonly KeyBinding WaitForAuthor = new([KeyPattern.Plain(ConsoleKey.W)], "w", "wait for author");
    public static readonly KeyBinding ResetVote = new([KeyPattern.Plain(ConsoleKey.D0)], "0", "reset vote");
    public static readonly KeyBinding Submit = new([KeyPattern.WithCtrl(ConsoleKey.S)], "ctrl+s", "submit");

    // PR lifecycle
    public static readonly KeyBinding Merge = new([KeyPattern.Plain(ConsoleKey.M)], "m", "merge PR");
    public static readonly KeyBinding Abandon = new([KeyPattern.WithShift(ConsoleKey.X)], "X", "abandon PR");
    public static readonly KeyBinding DraftToggle = new([KeyPattern.WithShift(ConsoleKey.D)], "D", "toggle draft/ready");
    public static readonly KeyBinding OpenBrowser = new([KeyPattern.Plain(ConsoleKey.O)], "o", "open in browser");

    // List
    public static readonly KeyBinding RepoPicker = new([KeyPattern.WithShift(ConsoleKey.R)], "R", "repo picker");
    public static readonly KeyBinding CreatePR = new([KeyPattern.Plain(ConsoleKey.N)], "n", "create PR");

    // Work items
    public static readonly KeyBinding StateTransition = new([KeyPattern.Plain(ConsoleKey.S)], "s", "state transition");
    public static readonly KeyBinding AddComment = new([KeyPattern.Plain(ConsoleKey.C)], "c", "add comment");
    public static readonly KeyBinding LinkPR = new([KeyPattern.WithShift(ConsoleKey.L)], "L", "link to PR");
}
