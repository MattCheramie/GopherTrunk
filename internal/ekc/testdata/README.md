# KFDtool key container test vector

`kfdtool_reference.ekc` was written by KFDtool's own container code, so the
reader is tested against the real format, not this package's idea of it
(issue #1305). Password: `gophertrunk-1305`. All keys are made up.

How it was made:

1. Copy, unmodified, `sw/control/KFDtool.Container/*.cs` and
   `sw/control/KFDtool.Shared/Utility.cs` from omahacommsys/KFDtool
   (commit fe4dcaf, MIT licence) into a .NET 8 console project.
2. Add the `System.Security.Cryptography.Xml` 8.0.2 package.
3. Use this `Program.cs`, then run `dotnet run -- kfdtool_reference.ekc gophertrunk-1305`.

It also reads the file back with KFDtool's `DecryptOuterContainer`, and checks
that a wrong password is rejected.

```csharp
// Generates a reference .ekc with KFDtool's own container code (unmodified
// sources in ref/), for GopherTrunk's decoder test vectors.
using System;
using System.Collections.Generic;
using System.IO;
using KFDtool.Container;

class Program
{
    static KeyItem K(int id, string name, int keyset, bool active, int sln, int keyId, int alg, string key, bool kek = false)
    {
        return new KeyItem { Id = id, Name = name, KeysetId = keyset, ActiveKeyset = active, Sln = sln,
            KeyTypeAuto = false, KeyTypeTek = !kek, KeyTypeKek = kek, KeyId = keyId, AlgorithmId = alg, Key = key };
    }

    static void Main(string[] args)
    {
        string password = args[1];
        InnerContainer inner = ContainerUtilities.CreateInnerContainer();
        inner.Keys.Add(K(1, "County AES", 1, true, 1, 1, 0x84, "000102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F"));
        inner.Keys.Add(K(2, "Fire DES", 1, true, 2, 0x2A, 0x81, "0123456789ABCDEF"));
        inner.Keys.Add(K(3, "Ops ADP", 1, true, 3, 3, 0xAA, "1234567890"));
        inner.Keys.Add(K(4, "Master KEK", 1, true, 4, 4, 0x84, "FFEEDDCCBBAA99887766554433221100FFEEDDCCBBAA99887766554433221100", kek: true));
        inner.Keys.Add(K(5, "County AES next", 2, false, 1, 1, 0x85, "A0A1A2A3A4A5A6A7A8A9AAABACADAEAF"));
        inner.Keys.Add(K(6, "Legacy DES-XL", 1, true, 6, 6, 0x9F, "0011223344556677"));
        inner.Keys.Add(K(7, "Ops TDES", 1, true, 7, 0x1234, 0x83, "00112233445566778899AABBCCDDEEFF0011223344556677"));
        inner.NextKeyNumber = 8;
        inner.Groups.Add(new GroupItem { Id = 1, Name = "All", Keys = new List<int> { 1, 2, 3 } });
        inner.NextGroupNumber = 2;
        var outer = ContainerUtilities.CreateOuterContainer(password);
        byte[] file = ContainerUtilities.EncryptOuterContainer(outer.Container, inner, outer.Key);
        File.WriteAllBytes(args[0], file);
        // Round trip through KFDtool's own reader.
        var back = ContainerUtilities.DecryptOuterContainer(file, password);
        Console.WriteLine($"wrote {args[0]}: {file.Length} bytes, {back.ContainerInner.Keys.Count} keys read back");
        try { ContainerUtilities.DecryptOuterContainer(file, password + "x"); Console.WriteLine("wrong password ACCEPTED"); }
        catch (Exception e) { Console.WriteLine("wrong password: " + e.Message); }
    }
}
```
