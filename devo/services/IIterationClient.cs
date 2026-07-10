using devo.models.pullrequests;

namespace devo.services;

public interface IIterationClient
{
    Task<IReadOnlyList<Iteration>> GetPullRequestIterationsAsync(string repoID, int prID, CancellationToken ct = default);
    Task<IReadOnlyList<IterationChange>> GetPullRequestIterationChangesAsync(string repoID, int prID, int iterationID, CancellationToken ct = default);
    Task<string> GetFileContentAtCommitAsync(string repoID, string filePath, string commitID, CancellationToken ct = default);
    Task<string> BuildUnifiedDiffAsync(string repoID, IterationChange change, string oldCommitID, string newCommitID, CancellationToken ct = default);
}