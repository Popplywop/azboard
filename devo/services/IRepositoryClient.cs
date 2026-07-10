using devo.models;
using devo.models.pullrequests;

namespace devo.services;

public interface IRepositoryClient
{
    Task<string> GetProjectIDAsync(CancellationToken ct = default);
    Task<IReadOnlyList<GitRepository>> ListRepositoriesAsync(CancellationToken ct = default);
    Task<IReadOnlyList<GitBranch>> ListBranchesAsync(string repoName, CancellationToken ct = default);
}