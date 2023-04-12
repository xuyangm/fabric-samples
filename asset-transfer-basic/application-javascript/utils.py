import zfec
import hashlib


def encode(k, n, m):
    """Erasure encodes string ``m`` into ``n`` blocks, such that any ``k``
    can reconstruct.
    :param int k: k
    :param int n: number of blocks to encode string ``m`` into.
    :param bytes m: bytestring to encode.
    :return list: Erasure codes resulting from encoding ``m`` into
        ``n`` blocks using ``zfec`` lib.
    """
    try:
        m = m.encode()
    except AttributeError:
        pass
    encoder = zfec.Encoder(k, n)
    assert k <= 256
    # pad m to a multiple of K bytes
    padlen = k - (len(m) % k)
    m += padlen * chr(k - padlen).encode()
    step = len(m) // k
    blocks = [m[i * step : (i + 1) * step] for i in range(k)]
    stripes = encoder.encode(blocks)
    return stripes


def decode(k, n, stripes):
    """Decodes an erasure-encoded string from a subset of stripes
    :param list stripes: a container of :math:`n` elements,
        each of which is either a string or ``None``
        at least :math:`k` elements are strings
        all string elements are the same length
    """
    assert len(stripes) == n
    blocks = []
    blocknums = []
    for i, block in enumerate(stripes):
        if block is None:
            continue
        blocks.append(block)
        blocknums.append(i)
        if len(blocks) == k:
            break
    else:
        raise ValueError("Too few to recover")
    decoder = zfec.Decoder(k, n)
    rec = decoder.decode(blocks, blocknums)
    m = b"".join(rec)
    padlen = k - m[-1]
    m = m[:-padlen]
    return m


def get_hash(x):
    assert isinstance(x, (str, bytes))
    try:
        x = x.encode()
    except AttributeError:
        pass
    return hashlib.sha256(x).hexdigest()