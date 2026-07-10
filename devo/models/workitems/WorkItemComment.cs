using System.Text.Json.Serialization;

using devo.models.pullrequests;

namespace devo.models.workitems;

/// <summary>A comment on a work item.</summary>
public sealed record WorkItemComment
{
    [JsonPropertyName("id")]
    public int ID { get; init; }

    [JsonPropertyName("text")]
    public required string Text { get; init; }

    [JsonPropertyName("createdBy")]
    public required IdentityRef CreatedBy { get; init; }

    [JsonPropertyName("createdDate")]
    public DateTime CreatedDate { get; init; }
}