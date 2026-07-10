using System.Text.Json.Serialization;

namespace devo.models.workitems;

/// <summary>Response for listing work item comments.</summary>
public sealed record WorkItemCommentsResult
{
    [JsonPropertyName("count")]
    public int Count { get; init; }

    [JsonPropertyName("comments")]
    public IReadOnlyList<WorkItemComment> Comments { get; init; } = [];
}