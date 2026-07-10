using devo.exceptions;

namespace devo.tests;

// port of internal/api/errors.go predicate semantics
public class ApiExceptionTests
{
    [Fact]
    public void Message_IncludesStatusAndBody()
    {
        var ex = new ApiException(403, "forbidden");
        Assert.Equal("API returned 403: forbidden", ex.Message);
    }

    [Theory]
    [InlineData(404, true, false, false, false)]
    [InlineData(401, false, true, false, false)]
    [InlineData(429, false, false, true, false)]
    [InlineData(500, false, false, false, true)]
    [InlineData(503, false, false, false, true)]
    [InlineData(200, false, false, false, false)]
    public void Predicates(int status, bool notFound, bool unauthorized, bool rateLimited, bool serverError)
    {
        var ex = new ApiException(status, "");
        Assert.Equal(notFound, ex.IsNotFound);
        Assert.Equal(unauthorized, ex.IsUnauthorized);
        Assert.Equal(rateLimited, ex.IsRateLimited);
        Assert.Equal(serverError, ex.IsServerError);
    }
}