namespace devo.services;

/// <summary>Allows the UI layer to clear cached data.</summary>
public interface ICacheInvalidator
{
    void InvalidateAll();
    void InvalidatePrefix(string prefix);
}