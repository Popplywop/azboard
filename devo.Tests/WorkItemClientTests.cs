using System.Text.Json;

using devo.models.workitems;
using devo.services;

namespace devo.tests;

// port of internal/api/workitems_test.go
public class WorkItemClientTests
{
    // --- WIQL query builder ---

    [Fact]
    public void BuildWiqlQuery_Basic()
    {
        string q = AdoClient.BuildWiqlQuery("MyProject", [], null, null, activeOnly: false);
        Assert.Contains("[System.TeamProject] = 'MyProject'", q);
        Assert.StartsWith("SELECT [System.Id] FROM WorkItems WHERE", q);
    }

    [Fact]
    public void BuildWiqlQuery_WithTypes()
    {
        string q = AdoClient.BuildWiqlQuery("P", ["Bug", "Task"], null, null, activeOnly: false);
        Assert.Contains("[System.WorkItemType] IN ('Bug', 'Task')", q);
    }

    [Fact]
    public void BuildWiqlQuery_WithAssignedTo()
    {
        string q = AdoClient.BuildWiqlQuery("P", [], "@me", null, activeOnly: false);
        Assert.Contains("[System.AssignedTo] = @me", q);
    }

    [Fact]
    public void BuildWiqlQuery_WithAreaPath()
    {
        string q = AdoClient.BuildWiqlQuery("P", [], null, "PDI\\Team", activeOnly: false);
        Assert.Contains("[System.AreaPath] UNDER 'PDI\\Team'", q);
    }

    [Fact]
    public void BuildWiqlQuery_ActiveOnly()
    {
        string q = AdoClient.BuildWiqlQuery("P", [], null, null, activeOnly: true);
        Assert.Contains("[System.State] NOT IN", q);
    }

    [Fact]
    public void BuildWiqlQuery_EscapesQuotes()
    {
        string q = AdoClient.BuildWiqlQuery("My'Project", ["User's Story"], null, "Path'Here", activeOnly: false);
        Assert.Contains("My''Project", q);
        Assert.Contains("User''s Story", q);
        Assert.Contains("Path''Here", q);
    }

    [Fact]
    public void BuildWiqlQuery_AllClauses()
    {
        string q = AdoClient.BuildWiqlQuery("P", ["Bug"], "@me", "Area", activeOnly: true);
        int whereIdx = q.IndexOf("WHERE ", StringComparison.Ordinal) + "WHERE ".Length;
        int orderIdx = q.IndexOf(" ORDER", StringComparison.Ordinal);
        string[] clauses = q[whereIdx..orderIdx].Split(" AND ");
        Assert.Equal(5, clauses.Length);
    }

    [Theory]
    [InlineData("normal", "normal")]
    [InlineData("it's", "it''s")]
    [InlineData("'quoted'", "''quoted''")]
    [InlineData("", "")]
    public void EscapeWiql(string input, string want) =>
        Assert.Equal(want, AdoClient.EscapeWiql(input));

    // --- HTTP integration ---

    [Fact]
    public async Task ListWorkItems()
    {
        var handler = new FakeHandler();
        handler.Handle("/wit/wiql", TestServer.Json(200, new
        {
            workItems = new[]
            {
                new { id = 1, url = "http://example.com/1" },
                new { id = 2, url = "http://example.com/2" },
            },
        }));
        handler.Handle("/wit/workitems", TestServer.Json(200, TestData.ListResponse(
            WorkItemJson(1, "Bug 1", "Bug"),
            WorkItemJson(2, "Story 1", "User Story"))));
        AdoClient client = TestServer.Client(handler);

        IReadOnlyList<WorkItem> items = await client.ListWorkItemsAsync(
            ["Bug", "User Story"], null, null, activeOnly: false);

        Assert.Equal(2, items.Count);

        CapturedRequest wiqlReq = handler.Requests[0];
        Assert.Equal(HttpMethod.Post, wiqlReq.Method);
        using JsonDocument wiqlBody = JsonDocument.Parse(wiqlReq.Body);
        Assert.Contains("System.TeamProject", wiqlBody.RootElement.GetProperty("query").GetString());

        Assert.Equal("1,2", handler.Requests[1].Query["ids"]);
    }

    [Fact]
    public async Task ListWorkItems_EmptyWiqlResult_ReturnsEmpty()
    {
        var handler = new FakeHandler();
        handler.Handle("/wit/wiql", TestServer.Json(200, new { workItems = Array.Empty<object>() }));
        AdoClient client = TestServer.Client(handler);

        IReadOnlyList<WorkItem> items = await client.ListWorkItemsAsync(["Bug"], null, null, activeOnly: false);

        Assert.Empty(items);
        Assert.Single(handler.Requests); // details endpoint never called
    }

    [Fact]
    public async Task GetWorkItem()
    {
        var handler = new FakeHandler();
        handler.Handle("/wit/workitems/42", TestServer.Json(200, WorkItemJson(42, "Test Item", "Bug")));
        AdoClient client = TestServer.Client(handler);

        WorkItem wi = await client.GetWorkItemAsync(42);

        Assert.Equal(42, wi.ID);
        Assert.Equal("Test Item", wi.Fields.Title);
        Assert.Equal("all", handler.Requests[0].Query["$expand"]);
    }

    [Fact]
    public async Task GetWorkItemComments()
    {
        var handler = new FakeHandler();
        handler.Handle("/wit/workitems/42/comments", TestServer.Json(200, new
        {
            count = 1,
            comments = new[]
            {
                new { id = 1, text = "hello", createdBy = TestData.Identity(), createdDate = "2024-01-01T00:00:00Z" },
            },
        }));
        AdoClient client = TestServer.Client(handler);

        IReadOnlyList<WorkItemComment> comments = await client.GetWorkItemCommentsAsync(42);

        Assert.Single(comments);
        Assert.Equal("hello", comments[0].Text);
        Assert.Equal("7.1-preview.4", handler.Requests[0].Query["api-version"]);
    }

    [Fact]
    public async Task UpdateWorkItemState()
    {
        var handler = new FakeHandler();
        handler.Handle("/wit/workitems/42", TestServer.Status(200));
        AdoClient client = TestServer.Client(handler);

        await client.UpdateWorkItemStateAsync(42, "Active");

        CapturedRequest req = handler.Requests[0];
        Assert.Equal(HttpMethod.Patch, req.Method);
        Assert.Equal("application/json-patch+json", req.ContentType);
        using JsonDocument ops = JsonDocument.Parse(req.Body);
        Assert.Equal(1, ops.RootElement.GetArrayLength());
        Assert.Equal("/fields/System.State", ops.RootElement[0].GetProperty("path").GetString());
        Assert.Equal("Active", ops.RootElement[0].GetProperty("value").GetString());
    }

    [Fact]
    public async Task AddWorkItemComment()
    {
        var handler = new FakeHandler();
        handler.Handle("/wit/workitems/42/comments", TestServer.Status(200));
        AdoClient client = TestServer.Client(handler);

        await client.AddWorkItemCommentAsync(42, "new comment");

        CapturedRequest req = handler.Requests[0];
        Assert.Equal(HttpMethod.Post, req.Method);
        Assert.Equal("7.1-preview.4", req.Query["api-version"]);
        using JsonDocument body = JsonDocument.Parse(req.Body);
        Assert.Equal("new comment", body.RootElement.GetProperty("text").GetString());
    }

    [Fact]
    public async Task GetWorkItemTypeStates()
    {
        var handler = new FakeHandler();
        handler.Handle("/wit/workitemtypes/", TestServer.Json(200, TestData.ListResponse(
            new { name = "New", color = "111", category = "Proposed" },
            new { name = "Active", color = "222", category = "InProgress" },
            new { name = "Closed", color = "333", category = "Completed" })));
        AdoClient client = TestServer.Client(handler);

        IReadOnlyList<string> states = await client.GetWorkItemTypeStatesAsync("Bug");

        Assert.Equal(["New", "Active", "Closed"], states);
    }

    [Fact]
    public async Task LinkWorkItemToPR()
    {
        var handler = new FakeHandler();
        handler.Handle("/wit/workitems/42", TestServer.Status(200));
        AdoClient client = TestServer.Client(handler);

        await client.LinkWorkItemToPRAsync(42, "vstfs:///Git/PullRequestId/proj/repo/5");

        CapturedRequest req = handler.Requests[0];
        Assert.Equal(HttpMethod.Patch, req.Method);
        using JsonDocument ops = JsonDocument.Parse(req.Body);
        Assert.Equal(1, ops.RootElement.GetArrayLength());
        Assert.Equal("/relations/-", ops.RootElement[0].GetProperty("path").GetString());
        JsonElement value = ops.RootElement[0].GetProperty("value");
        Assert.Equal("ArtifactLink", value.GetProperty("rel").GetString());
        Assert.Equal("Pull Request", value.GetProperty("attributes").GetProperty("name").GetString());
    }

    private static object WorkItemJson(int id, string title, string type) => new
    {
        id,
        url = $"http://example.com/{id}",
        fields = new Dictionary<string, object>
        {
            ["System.Title"] = title,
            ["System.State"] = "Active",
            ["System.WorkItemType"] = type,
        },
    };
}