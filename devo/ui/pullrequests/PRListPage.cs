using ConsoleForge.Core;
using ConsoleForge.Layout;
using ConsoleForge.Widgets;

using devo.exceptions;
using devo.models.pullrequests;
using devo.services;

namespace devo.ui.pullrequests;

[DispatchUpdate]
internal sealed partial record PRListPage(
    IAdoClient AdoClient,
    IReadOnlyList<string> Repositories) : IComponent
{
    public IReadOnlyList<PullRequest> PRs { get; init; } = [];
    public int Selected { get; init; }
    public bool Loading { get; init; } = true;
    public string? Error { get; init; }
    public int SpinnerFrame { get; init; }

    private static readonly KeyMap Keys = new KeyMap()
        .On(KeyBindings.Up, () => new NavUpMsg())
        .On(KeyBindings.Down, () => new NavDownMsg())
        .On(KeyBindings.Refresh, () => new RefreshMsg());

    private static ICmd SpinnerTick() =>
        Cmd.Tick(TimeSpan.FromMilliseconds(120), at => new TickMsg(at));

    public ICmd? Init() => Cmd.Batch(SpinnerTick(), FetchPRsCmd());

    private (IModel Model, ICmd? Cmd) OnPRsLoaded(PRsLoadedMsg msg) =>
        (this with { PRs = msg.PRs, Loading = false, Error = null }, null);

    private (IModel Model, ICmd? Cmd) OnPRsError(PRsErrorMsg msg) =>
        (this with { Error = msg.Error.Message, Loading = false }, null);

    private (IModel Model, ICmd? Cmd) OnNavUp() =>
        (this with { Selected = Math.Max(0, Selected - 1) }, null);

    private (IModel Model, ICmd? Cmd) OnNavDown() =>
        (this with { Selected = Math.Min(Math.Max(0, PRs.Count - 1), Selected + 1) }, null);

    private (IModel Model, ICmd? Cmd) OnRefresh() =>
        (this with { Loading = true, Error = null }, Cmd.Batch(FetchPRsCmd(), SpinnerTick()));

    // re-arm only while loading — the chain ends quietly once content is up
    private (IModel Model, ICmd? Cmd) OnTick() =>
        Loading
            ? (this with { SpinnerFrame = SpinnerFrame + 1 }, SpinnerTick())
            : (this, null);

    public IWidget View()
    {
        if (Loading)
        {
            const string label = "Please wait…";
            return Centered(new Spinner(SpinnerFrame, label: label, frames: Spinner.ArcFrames)
            {
                Width = SizeConstraint.Fixed(label.Length + 2), // frame glyph + space + label
                Height = SizeConstraint.Fixed(1),
            });
        }

        if (Error is not null)
        {
            return new TextBlock($"Error: {Error}\n\nPress r to retry");
        }
        if (PRs.Count == 0)
        {
            return new TextBlock("No active pull requests. Press r to refresh.");
        }

        TableColumn[] columns =
        [
            new("#", 6),
            new("Title", 40),
            new("Repository", 20),
            new("Author", 18),
            new("Status", 10),
            new("Reviewers", 24),
        ];
        List<IReadOnlyList<string>> rows = [.. PRs
            .Select(pr => (IReadOnlyList<string>)
            [
                pr.PullRequestID.ToString(),
                pr.IsDraft ? $"[draft] {pr.Title}" : pr.Title,
                pr.Repository.Name,
                pr.CreatedBy.DisplayName,
                pr.Status,
                string.Join(" ", pr.Reviewers.Select(r => r.VoteIcon())),
            ])];

        return new Table(columns, rows, Selected);
    }

    /// <summary>Centers a widget using flex spacers. The inner widget MUST
    /// declare Fixed Width/Height: the layout engine treats Auto as Flex(1)
    /// (no content measurement), so flex spacers only center around fixed
    /// children.</summary>
    private static Container Centered(IWidget inner, int rowHeight = 1) =>
        new(Axis.Vertical,
        [
            new TextBlock(string.Empty) { Height = SizeConstraint.Flex(1) },
            new Container(Axis.Horizontal,
            [
                new TextBlock(string.Empty) { Width = SizeConstraint.Flex(1) },
                inner,
                new TextBlock(string.Empty) { Width = SizeConstraint.Flex(1) },
            ])
            { Height = SizeConstraint.Fixed(rowHeight) },
            new TextBlock(string.Empty) { Height = SizeConstraint.Flex(1) },
        ]);

    /// <summary>Fetches PRs for the configured repos concurrently, skipping
    /// repos that 404. No repos configured = project-wide query.</summary>
    private ICmd FetchPRsCmd()
    {
        IAdoClient client = AdoClient;
        IReadOnlyList<string> repos = Repositories;
        return Cmd.Run(async ct =>
        {
            try
            {
                if (repos.Count == 0)
                {
                    IReadOnlyList<PullRequest> all = await client.ListPullRequestsAsync("active", ct);
                    return new PRsLoadedMsg(all);
                }

                IReadOnlyList<PullRequest>[] results = await Task.WhenAll(
                    repos.Select(async repo =>
                    {
                        try
                        {
                            return await client.ListPullRequestsForRepoAsync(repo, "active", ct);
                        }
                        catch (ApiException e) when (e.IsNotFound)
                        {
                            return []; // repo renamed/removed — skip
                        }
                    }));

                List<PullRequest> merged = [.. results
                    .SelectMany(r => r)
                    .OrderByDescending(pr => pr.CreationDate)];
                return new PRsLoadedMsg(merged);
            }
            catch (Exception e)
            {
                return new PRsErrorMsg(e);
            }
        });
    }

    private sealed record PRsLoadedMsg(IReadOnlyList<PullRequest> PRs) : IMsg;
    private sealed record PRsErrorMsg(Exception Error) : IMsg;
    private sealed record NavUpMsg : IMsg;
    private sealed record NavDownMsg : IMsg;
    private sealed record RefreshMsg : IMsg;
    private sealed record TickMsg(DateTimeOffset At) : IMsg;
}
