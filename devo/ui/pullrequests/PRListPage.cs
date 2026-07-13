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

    private static readonly KeyMap Keys = new KeyMap()
        .On(KeyBindings.Up, () => new NavUpMsg())
        .On(KeyBindings.Down, () => new NavDownMsg())
        .On(KeyBindings.Refresh, () => new RefreshMsg());

    public ICmd? Init() => FetchPRsCmd();

    private (IModel Model, ICmd? Cmd) OnPRsLoaded(PRsLoadedMsg msg) =>
        (this with { PRs = msg.PRs, Loading = false, Error = null }, null);

    private (IModel Model, ICmd? Cmd) OnPRsError(PRsErrorMsg msg) =>
        (this with { Error = msg.Error.Message, Loading = false }, null);

    private (IModel Model, ICmd? Cmd) OnNavUp() =>
        (this with { Selected = Math.Max(0, Selected - 1) }, null);

    private (IModel Model, ICmd? Cmd) OnNavDown() =>
        (this with { Selected = Math.Min(Math.Max(0, PRs.Count - 1), Selected + 1) }, null);

    private (IModel Model, ICmd? Cmd) OnRefresh() =>
        (this with { Loading = true, Error = null }, FetchPRsCmd());

    public IWidget View()
    {
        if (Loading)
        {
            return new TextBlock("Loading pull requests…");
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

                List<PullRequest> merged = results
                    .SelectMany(r => r)
                    .OrderByDescending(pr => pr.CreationDate)
                    .ToList();
                return new PRsLoadedMsg(merged);
            }
            catch (Exception e)
            {
                return (IMsg)new PRsErrorMsg(e);
            }
        });
    }

    private sealed record PRsLoadedMsg(IReadOnlyList<PullRequest> PRs) : IMsg;
    private sealed record PRsErrorMsg(Exception Error) : IMsg;
    private sealed record NavUpMsg : IMsg;
    private sealed record NavDownMsg : IMsg;
    private sealed record RefreshMsg : IMsg;
}
