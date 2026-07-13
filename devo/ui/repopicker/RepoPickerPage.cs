using ConsoleForge.Core;
using ConsoleForge.Layout;
using ConsoleForge.Widgets;

using devo.services;

namespace devo.ui.repopicker;

/// <summary>Floating repo selection modal — port of internal/ui/repopicker.
/// Completes with a PickerResult; parent checks Component.IsCompleted.</summary>
[DispatchUpdate]
internal sealed partial record RepoPickerPage(
    IAdoClient AdoClient,
    IReadOnlyList<string> InitialSelected) : IComponent<PickerResult>
{
    public IReadOnlyList<string> Repositories { get; init; } = [];
    public IReadOnlySet<string> Selected { get; init; } = new HashSet<string>(InitialSelected);
    public int Cursor { get; init; }
    public string FilterText { get; init; } = string.Empty;
    public bool Filtering { get; init; }
    public bool Loading { get; init; } = true;
    public string? Error { get; init; }
    public PickerResult? Result { get; init; }
    public int Width { get; init; } = 80;
    public int Height { get; init; } = 24;

    PickerResult IComponent<PickerResult>.Result => Result!;

    // port of picker.go View sizing: ~60% of terminal, floor 50x10, cap width-4
    private int BoxWidth => Math.Min(Math.Max(Width * 60 / 100, 50), Math.Max(1, Width - 4));
    private int BoxHeight => Math.Max(Height * 60 / 100, 10);
    // dialog minus title, filter line, spacers, help line, border
    private int ListRows => Math.Max(1, BoxHeight - 6);

    /// <summary>Repositories matching the current filter text.</summary>
    private IReadOnlyList<string> Filtered =>
        FilterText.Length == 0
            ? Repositories
            : [.. Repositories.Where(r => r.Contains(FilterText.Trim(), StringComparison.OrdinalIgnoreCase))];

    private int ClampedCursor => Math.Clamp(Cursor, 0, Math.Max(0, Filtered.Count - 1));

    public ICmd? Init() => FetchReposCmd();

    private (IModel Model, ICmd? Cmd) OnReposLoaded(ReposLoadedMsg msg) =>
        (this with { Repositories = msg.Repos, Loading = false, Error = null, Cursor = 0 }, null);

    private (IModel Model, ICmd? Cmd) OnReposLoadError(ReposLoadErrorMsg msg) =>
        (this with { Error = msg.Error.Message, Loading = false }, null);

    private (IModel Model, ICmd? Cmd) OnWindowResize(WindowResizeMsg msg) =>
        (this with { Width = msg.Width, Height = msg.Height }, null);

    private (IModel Model, ICmd? Cmd) OnKey(KeyMsg msg) =>
        Filtering ? HandleFilteringKey(msg) : HandleNormalKey(msg);

    private (IModel Model, ICmd? Cmd) HandleFilteringKey(KeyMsg key)
    {
        return key switch
        {
            { Ctrl: true, Key: ConsoleKey.C } => Cancel(),
            { Key: ConsoleKey.Escape } => FilterText.Length > 0
                                ? (this with { FilterText = string.Empty, Cursor = 0 }, null)
                                : (this with { Filtering = false }, null),// first esc clears the filter text, second leaves filter mode
            { Key: ConsoleKey.Enter } => (this with { Filtering = false }, null),
            { Key: ConsoleKey.Backspace } => FilterText.Length > 0
                                ? (this with { FilterText = FilterText[..^1], Cursor = 0 }, null)
                                : (this, null),
            { Character: { } c } when !char.IsControl(c) => (this with { FilterText = FilterText + c, Cursor = 0 }, null),
            _ => (this, null),
        };

    }

    private (IModel Model, ICmd? Cmd) HandleNormalKey(KeyMsg key)
    {
        return key switch
        {
            { Ctrl: true, Key: ConsoleKey.C } => Cancel(),
            { Ctrl: true, Key: ConsoleKey.S } => Done(save: true),
            { Key: ConsoleKey.Escape } => FilterText.Length > 0
                                ? (this with { FilterText = string.Empty, Cursor = 0 }, null)
                                : Cancel(),
            { Key: ConsoleKey.Enter } => Done(save: false),
            { Key: ConsoleKey.UpArrow } or { Character: 'k' } => (this with { Cursor = Math.Max(0, ClampedCursor - 1) }, null),
            { Key: ConsoleKey.DownArrow } or { Character: 'j' } => (this with { Cursor = Math.Min(Math.Max(0, Filtered.Count - 1), ClampedCursor + 1) }, null),
            { Key: ConsoleKey.Spacebar } or { Character: ' ' } => (ToggleAtCursor(), null),
            { Character: '/' } => (this with { Filtering = true }, null),
            _ => (this, null),
        };

    }

    private RepoPickerPage ToggleAtCursor()
    {
        IReadOnlyList<string> filtered = Filtered;
        if (filtered.Count == 0)
        {
            return this;
        }
        string name = filtered[ClampedCursor];
        HashSet<string> next = [.. Selected];
        if (!next.Add(name))
        {
            next.Remove(name);
        }
        return this with { Selected = next };
    }

    private (IModel Model, ICmd? Cmd) Done(bool save) =>
        (this with { Result = new PickerResult([.. Selected.Order()], save, Cancelled: false) }, null);

    private (IModel Model, ICmd? Cmd) Cancel() =>
        (this with { Result = new PickerResult([], Save: false, Cancelled: true) }, null);

    public IWidget View()
    {
        IWidget body;
        if (Loading)
        {
            body = new TextBlock("Loading repositories...");
        }
        else if (Error is not null)
        {
            body = new TextBlock($"Error: {Error}");
        }
        else
        {
            IReadOnlyList<string> filtered = Filtered;
            // Go's scroll window: cursor stays on the last visible row once
            // it moves past the window (picker.go:250-258)
            int cursor = ClampedCursor;
            int scrollStart = cursor >= ListRows ? cursor - ListRows + 1 : 0;
            IWidget list = filtered.Count == 0
                ? new TextBlock("No repositories match.")
                : new List(
                    [.. filtered.Select(r => $"[{(Selected.Contains(r) ? 'x' : ' ')}] {r}")],
                    cursor)
                {
                    ScrollOffset = scrollStart,
                    Height = SizeConstraint.Fixed(ListRows),
                };
            string prompt = Filtering ? $"Search: {FilterText}_" : $"Search: {FilterText}";
            body = new Container(Axis.Vertical,
            [
                new TextBlock(prompt),
                list,
                new TextBlock("space toggle · / filter · enter confirm · ctrl+s save & confirm · esc cancel"),
            ]);
        }

        // no backdrop: ZStack shows the PR list around the dialog
        return new Modal("Select Repositories", body, BoxWidth, BoxHeight, showBackdrop: false);
    }

    private ICmd FetchReposCmd()
    {
        IAdoClient client = AdoClient;
        return Cmd.Run(async ct =>
        {
            try
            {
                var repos = await client.ListRepositoriesAsync(ct);
                return new ReposLoadedMsg([.. repos.Select(r => r.Name).Order()]);
            }
            catch (Exception e)
            {
                return (IMsg)new ReposLoadErrorMsg(e);
            }
        });
    }

    private sealed record ReposLoadedMsg(IReadOnlyList<string> Repos) : IMsg;
    private sealed record ReposLoadErrorMsg(Exception Error) : IMsg;
}
