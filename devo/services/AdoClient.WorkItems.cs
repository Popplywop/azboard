using devo.models;
using devo.models.requests;
using devo.models.workitems;

namespace devo.services;

public sealed partial class AdoClient
{
    /// <summary>Queries work items via WIQL and fetches their details.
    /// assignedTo is a WIQL keyword like "@me"; null = no filter.
    /// areaPath restricts to items under the given area path; null = no filter.</summary>
    public async Task<IReadOnlyList<WorkItem>> ListWorkItemsAsync(IReadOnlyList<string> types, string? assignedTo, string? areaPath, bool activeOnly, CancellationToken ct = default)
    {
        string query = BuildWiqlQuery(_project, types, assignedTo, areaPath, activeOnly);

        // $top=50 bounds initial page size
        WiqlResult result = await PostAsync<WiqlResult>(
            "/wit/wiql?$top=50", new WiqlRequest { Query = query }, ct);

        if (result.WorkItems.Count == 0)
        {
            return [];
        }

        // max 200 IDs per request per ADO limits
        string ids = string.Join(",", result.WorkItems.Take(200).Select(r => r.ID));
        ListResponse<WorkItem> resp = await GetAsync<ListResponse<WorkItem>>(
            $"/wit/workitems?ids={ids}&$expand=fields", ct);
        return resp.Value;
    }

    public Task<WorkItem> GetWorkItemAsync(int id, CancellationToken ct = default) =>
        GetAsync<WorkItem>($"/wit/workitems/{id}?$expand=all", ct);

    public async Task<IReadOnlyList<WorkItemComment>> GetWorkItemCommentsAsync(int id, CancellationToken ct = default)
    {
        WorkItemCommentsResult resp = await GetPreviewAsync<WorkItemCommentsResult>(
            $"/wit/workitems/{id}/comments", "7.1-preview.4", ct);
        return resp.Comments;
    }

    public Task UpdateWorkItemStateAsync(int id, string state, CancellationToken ct = default)
    {
        WorkItemPatchOp[] body =
        [
            new() { Op = "add", Path = "/fields/System.State", Value = state },
        ];
        return PatchJsonPatchAsync($"/wit/workitems/{id}", body, ct);
    }

    public Task AddWorkItemCommentAsync(int id, string text, CancellationToken ct = default) =>
        PostPreviewAsync($"/wit/workitems/{id}/comments", "7.1-preview.4",
            new AddWorkItemCommentRequest { Text = text }, ct);

    /// <summary>Fetches valid states for a work item type. Callers fall back
    /// to WorkItemStates.Fallback if this throws.</summary>
    public async Task<IReadOnlyList<string>> GetWorkItemTypeStatesAsync(string workItemType, CancellationToken ct = default)
    {
        ListResponse<WorkItemTypeState> resp = await GetAsync<ListResponse<WorkItemTypeState>>(
            $"/wit/workitemtypes/{Uri.EscapeDataString(workItemType)}/states", ct);
        return resp.Value.Select(s => s.Name).ToList();
    }

    /// <summary>Links a work item to a PR via an artifact link.
    /// prArtifactURL format: vstfs:///Git/PullRequestId/{projectID}/{repoID}/{prID}</summary>
    public Task LinkWorkItemToPRAsync(int workItemID, string prArtifactURL, CancellationToken ct = default)
    {
        WorkItemPatchOp[] body =
        [
            new()
            {
                Op = "add",
                Path = "/relations/-",
                Value = new WorkItemLinkValue
                {
                    Rel = "ArtifactLink",
                    URL = prArtifactURL,
                    Attributes = new Dictionary<string, object> { ["name"] = "Pull Request" },
                },
            },
        ];
        return PatchJsonPatchAsync($"/wit/workitems/{workItemID}", body, ct);
    }

    /// <summary>Escapes single quotes for WIQL string literals.</summary>
    internal static string EscapeWiql(string s) => s.Replace("'", "''");

    internal static string BuildWiqlQuery(string project, IReadOnlyList<string> types, string? assignedTo, string? areaPath, bool activeOnly)
    {
        var clauses = new List<string>
        {
            $"[System.TeamProject] = '{EscapeWiql(project)}'",
        };

        if (types.Count > 0)
        {
            string quoted = string.Join(", ", types.Select(t => "'" + EscapeWiql(t) + "'"));
            clauses.Add($"[System.WorkItemType] IN ({quoted})");
        }

        if (!string.IsNullOrEmpty(assignedTo))
        {
            // assignedTo is a WIQL keyword like @me, not a string literal
            clauses.Add($"[System.AssignedTo] = {assignedTo}");
        }

        if (!string.IsNullOrEmpty(areaPath))
        {
            clauses.Add($"[System.AreaPath] UNDER '{EscapeWiql(areaPath)}'");
        }

        if (activeOnly)
        {
            clauses.Add("[System.State] NOT IN ('Closed', 'Done', 'Resolved', 'Removed')");
        }

        return "SELECT [System.Id] FROM WorkItems WHERE "
            + string.Join(" AND ", clauses)
            + " ORDER BY [System.ChangedDate] DESC";
    }
}