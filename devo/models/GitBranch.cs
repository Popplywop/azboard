using System.Text.Json.Serialization;

namespace devo.models;

/// <summary>A git branch (ref) in Azure DevOps.</summary>
public sealed record GitBranch
{
    /// <summary>Full ref name, e.g. "refs/heads/main".</summary>
    [JsonPropertyName("name")]
    public required string Name { get; init; }

    [JsonIgnore]
    public string ShortName => RefName.StripPrefix(Name);
}