using System.Text.Json.Serialization;

using devo.models.pullrequests;

namespace devo.models.workitems;

/// <summary>System fields of a work item.</summary>
public sealed record WorkItemFields
{
    [JsonPropertyName("System.Title")]
    public required string Title { get; init; }

    [JsonPropertyName("System.State")]
    public required string State { get; init; }

    [JsonPropertyName("System.WorkItemType")]
    public required string WorkItemType { get; init; }

    [JsonPropertyName("System.AssignedTo")]
    public IdentityRef? AssignedTo { get; init; }

    [JsonPropertyName("System.Description")]
    public string? Description { get; init; }

    [JsonPropertyName("System.CreatedDate")]
    public DateTime CreatedDate { get; init; }

    [JsonPropertyName("System.ChangedDate")]
    public DateTime ChangedDate { get; init; }

    [JsonPropertyName("System.AreaPath")]
    public string? AreaPath { get; init; }

    [JsonPropertyName("System.TeamProject")]
    public string? TeamProject { get; init; }
}