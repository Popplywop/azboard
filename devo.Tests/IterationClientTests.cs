using devo.models.pullrequests;
using devo.services;

namespace devo.tests;

// port of internal/api/iterations_test.go + diff_test.go
public class IterationClientTests
{
    [Fact]
    public async Task GetPullRequestIterations()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories/repo-1/pullrequests/5/iterations",
            TestServer.Json(200, TestData.ListResponse(
                IterationJson(1, "abc", "def"),
                IterationJson(2, "ghi", "jkl"))));
        AdoClient client = TestServer.Client(handler);

        IReadOnlyList<Iteration> iters = await client.GetPullRequestIterationsAsync("repo-1", 5);

        Assert.Equal(HttpMethod.Get, handler.Requests[0].Method);
        Assert.Equal(2, iters.Count);
        Assert.Equal("abc", iters[0].SourceRefCommit.CommitID);
    }

    [Fact]
    public async Task GetPullRequestIterationChanges()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories/repo-1/pullrequests/5/iterations/1/changes",
            TestServer.Json(200, new
            {
                changeEntries = new[]
                {
                    new { changeTrackingId = 1, changeType = "edit", item = new { path = "/src/main.go" } },
                    new { changeTrackingId = 2, changeType = "add", item = new { path = "/src/new.go" } },
                },
            }));
        AdoClient client = TestServer.Client(handler);

        IReadOnlyList<IterationChange> changes = await client.GetPullRequestIterationChangesAsync("repo-1", 5, 1);

        Assert.Equal("2000", handler.Requests[0].Query["$top"]);
        Assert.Equal(2, changes.Count);
        Assert.Equal("edit", changes[0].ChangeType);
        Assert.Equal("/src/new.go", changes[1].Item.Path);
    }

    // --- diff helpers (port of diff_test.go) ---

    [Theory]
    [InlineData("a\r\nb\r\nc", "a\nb\nc")]   // CRLF
    [InlineData("a\rb\rc", "a\nb\nc")]       // CR
    [InlineData("a\nb\nc", "a\nb\nc")]       // LF passthrough
    [InlineData("a\r\nb\rc\n", "a\nb\nc\n")] // mixed
    public void NormalizeLineEndings(string input, string want) =>
        Assert.Equal(want, AdoClient.NormalizeLineEndings(input));

    [Fact]
    public void GenerateUnifiedDiff_SimpleEdit()
    {
        string diff = AdoClient.GenerateUnifiedDiff("a\nb\nc", "a\nx\nc", "a/f.txt", "b/f.txt");

        Assert.Equal(
            "--- a/f.txt\n" +
            "+++ b/f.txt\n" +
            "@@ -1,3 +1,3 @@\n" +
            " a\n" +
            "-b\n" +
            "+x\n" +
            " c\n",
            diff);
    }

    [Fact]
    public void GenerateUnifiedDiff_NoChanges_ReturnsEmpty()
    {
        string diff = AdoClient.GenerateUnifiedDiff("same\ncontent", "same\ncontent", "a/f", "b/f");
        Assert.Equal("", diff);
    }

    [Fact]
    public void GenerateUnifiedDiff_DistantChanges_SplitIntoHunks()
    {
        // two edits 20 lines apart -> two @@ hunks
        string[] lines = Enumerable.Range(1, 30).Select(i => $"line{i}").ToArray();
        string oldText = string.Join('\n', lines);
        lines[0] = "CHANGED-TOP";
        lines[29] = "CHANGED-BOTTOM";
        string newText = string.Join('\n', lines);

        string diff = AdoClient.GenerateUnifiedDiff(oldText, newText, "a/f", "b/f");

        Assert.Equal(2, diff.Split("@@ -").Length - 1);
        Assert.Contains("-line1\n", diff);
        Assert.Contains("+CHANGED-TOP\n", diff);
        Assert.Contains("-line30\n", diff);
        Assert.Contains("+CHANGED-BOTTOM\n", diff);
    }

    [Fact]
    public async Task BuildUnifiedDiff_RenamePrependsHeader()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories/repo-1/items", req =>
            TestServer.Text(200, "same\ncontent\n"));
        AdoClient client = TestServer.Client(handler);

        var change = new IterationChange
        {
            ChangeID = 1,
            ChangeType = "rename",
            Item = new ChangeItem { Path = "/new-name.txt" },
            OriginalPath = "/old-name.txt",
        };
        string diff = await client.BuildUnifiedDiffAsync("repo-1", change, "old-sha", "new-sha");

        Assert.StartsWith("rename from /old-name.txt\nrename to /new-name.txt\n", diff);
        Assert.Contains("(no textual changes)", diff);
    }

    [Fact]
    public async Task BuildUnifiedDiff_AddFetchesOnlyNewContent()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories/repo-1/items", req =>
            TestServer.Text(200, "brand new\n"));
        AdoClient client = TestServer.Client(handler);

        var change = new IterationChange
        {
            ChangeID = 1,
            ChangeType = "add",
            Item = new ChangeItem { Path = "/new.txt" },
        };
        await client.BuildUnifiedDiffAsync("repo-1", change, "old-sha", "new-sha");

        Assert.Single(handler.Requests); // old content never fetched
        Assert.Equal("new-sha", handler.Requests[0].Query["version"]);
    }

    private static object IterationJson(int id, string sourceCommit, string targetCommit) => new
    {
        id,
        description = "",
        createdDate = "2024-01-01T00:00:00Z",
        sourceRefCommit = new { commitId = sourceCommit },
        targetRefCommit = new { commitId = targetCommit },
    };
}