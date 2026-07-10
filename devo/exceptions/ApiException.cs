namespace devo.exceptions;

/// <summary>
/// Thrown when the Azure DevOps API returns a non-success status code.
/// </summary>
public sealed class ApiException : Exception
{
    public int StatusCode { get; }
    public string Body { get; }

    public ApiException(int statusCode, string body)
        : base($"API returned {statusCode}: {body}")
    {
        StatusCode = statusCode;
        Body = body;
    }

    public bool IsNotFound => StatusCode == 404;
    public bool IsUnauthorized => StatusCode == 401;
    public bool IsRateLimited => StatusCode == 429;
    public bool IsServerError => StatusCode >= 500;
}