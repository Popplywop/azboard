using System.Text.Json.Serialization;

namespace devo.models.pullrequests;

/// <summary>Identifies a git commit in Azure DevOps.</summary>
public sealed record CommitRef
{
    [JsonPropertyName("commitId")]
    public required string CommitID { get; init; }
}