using System.Text.Json.Serialization;

using devo.models;
using devo.models.pullrequests;

namespace devo.services;

public sealed partial class AdoClient
{
    public async Task<string> GetProjectIDAsync(CancellationToken ct = default)
    {
        ProjectResponse proj = await GetOrgAsync<ProjectResponse>($"/projects/{_project}", ct);
        return proj.ID;
    }

    public async Task<IReadOnlyList<GitRepository>> ListRepositoriesAsync(CancellationToken ct = default)
    {
        ListResponse<GitRepository> resp =
            await GetAsync<ListResponse<GitRepository>>("/git/repositories", ct);
        return resp.Value;
    }

    public async Task<IReadOnlyList<GitBranch>> ListBranchesAsync(string repoName, CancellationToken ct = default)
    {
        ListResponse<GitBranch> resp = await GetAsync<ListResponse<GitBranch>>(
            $"/git/repositories/{repoName}/refs?filter=heads&$top=1000&peelTags=false", ct);
        return resp.Value;
    }

    /// <summary>Minimal shape of an ADO project returned by the projects API.</summary>
    private sealed record ProjectResponse
    {
        [JsonPropertyName("id")]
        public required string ID { get; init; }

        [JsonPropertyName("name")]
        public string? Name { get; init; }
    }
}