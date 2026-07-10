using devo.models;
using devo.models.pullrequests;

namespace devo.tests;

// port of internal/api/types_test.go
public class ModelTests
{
    [Theory]
    [InlineData("refs/heads/main", "main")]
    [InlineData("refs/heads/feature/foo", "feature/foo")]
    [InlineData("refs/heads/a", "a")]
    [InlineData("refs/heads/", "")]
    public void StripPrefix_Valid(string input, string want) =>
        Assert.Equal(want, RefName.StripPrefix(input));

    [Theory]
    [InlineData("main")]
    [InlineData("feature/foo")]
    [InlineData("something-long-string")]
    [InlineData("")]
    public void StripPrefix_NoPrefix_PassesThrough(string input) =>
        Assert.Equal(input, RefName.StripPrefix(input));

    [Fact]
    public void PullRequest_SourceBranch()
    {
        PullRequest pr = MakePr(source: "refs/heads/feature-x");
        Assert.Equal("feature-x", pr.SourceBranch);
    }

    [Fact]
    public void PullRequest_TargetBranch()
    {
        PullRequest pr = MakePr(target: "refs/heads/main");
        Assert.Equal("main", pr.TargetBranch);
    }

    [Theory]
    [InlineData(10, "Approved")]
    [InlineData(5, "Approved with suggestions")]
    [InlineData(0, "No vote")]
    [InlineData(-5, "Waiting for author")]
    [InlineData(-10, "Rejected")]
    [InlineData(99, "Unknown")]
    public void Reviewer_VoteString(int vote, string want) =>
        Assert.Equal(want, MakeReviewer(vote).VoteString());

    // Go used "·" for no-vote; the C# port uses "." — test pins the C# behavior
    [Theory]
    [InlineData(10, "✓")]
    [InlineData(5, "~")]
    [InlineData(0, ".")]
    [InlineData(-5, "⏳")]
    [InlineData(-10, "✗")]
    [InlineData(99, "?")]
    public void Reviewer_VoteIcon(int vote, string want) =>
        Assert.Equal(want, MakeReviewer(vote).VoteIcon());

    [Fact]
    public void GitBranch_ShortName()
    {
        var branch = new GitBranch { Name = "refs/heads/feature-branch" };
        Assert.Equal("feature-branch", branch.ShortName);
    }

    private static PullRequest MakePr(string source = "refs/heads/feature", string target = "refs/heads/main") =>
        new()
        {
            Title = "t",
            Status = "active",
            SourceRefName = source,
            TargetRefName = target,
            CreatedBy = new IdentityRef { DisplayName = "d", UniqueName = "u", ID = "i" },
            Repository = new GitRepository { ID = "r", Name = "n" },
        };

    private static Reviewer MakeReviewer(int vote) =>
        new() { DisplayName = "d", UniqueName = "u", ID = "i", Vote = vote };
}