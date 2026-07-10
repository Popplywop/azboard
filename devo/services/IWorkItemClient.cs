using devo.models.workitems;

namespace devo.services;

public interface IWorkItemClient
{
    Task<IReadOnlyList<WorkItem>> ListWorkItemsAsync(IReadOnlyList<string> types, string? assignedTo, string? areaPath, bool activeOnly, CancellationToken ct = default);
    Task<WorkItem> GetWorkItemAsync(int id, CancellationToken ct = default);
    Task<IReadOnlyList<WorkItemComment>> GetWorkItemCommentsAsync(int id, CancellationToken ct = default);
    Task UpdateWorkItemStateAsync(int id, string state, CancellationToken ct = default);
    Task AddWorkItemCommentAsync(int id, string text, CancellationToken ct = default);
    Task<IReadOnlyList<string>> GetWorkItemTypeStatesAsync(string workItemType, CancellationToken ct = default);
    Task LinkWorkItemToPRAsync(int workItemID, string prArtifactURL, CancellationToken ct = default);
}