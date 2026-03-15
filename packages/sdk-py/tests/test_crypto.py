from __future__ import annotations

import base64
import json

from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import ec, x25519
from cryptography.hazmat.primitives.asymmetric.utils import decode_dss_signature
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
from cryptography.hazmat.primitives.serialization import load_der_public_key

from dginf_sdk.crypto import encrypt_job_envelope, verify_provider_session_key


def test_encrypt_job_envelope_round_trip() -> None:
    recipient_private = x25519.X25519PrivateKey.generate()
    recipient_public = recipient_private.public_key().public_bytes(
        encoding=serialization.Encoding.Raw,
        format=serialization.PublicFormat.Raw,
    )
    envelope = encrypt_job_envelope(
        base64.b64encode(recipient_public).decode("utf-8"),
        "hello world",
        12,
    )
    shared_secret = recipient_private.exchange(
        x25519.X25519PublicKey.from_public_bytes(base64.b64decode(envelope.ephemeral_pubkey))
    )
    digest = hashes.Hash(hashes.SHA256())
    digest.update(shared_secret)
    digest.update(b"dginf-envelope-v1")
    key = digest.finalize()
    plaintext = AESGCM(key).decrypt(
        base64.b64decode(envelope.nonce),
        base64.b64decode(envelope.ciphertext),
        None,
    )
    payload = json.loads(plaintext)
    assert payload["prompt"] == "hello world"
    assert payload["max_output_tokens"] == 12


def test_verify_provider_session_key() -> None:
    signing_key = ec.generate_private_key(ec.SECP256R1())
    signing_pub = signing_key.public_key().public_bytes(
        encoding=serialization.Encoding.DER,
        format=serialization.PublicFormat.SubjectPublicKeyInfo,
    )
    session_pub = base64.b64encode(x25519.X25519PrivateKey.generate().public_key().public_bytes(
        encoding=serialization.Encoding.Raw,
        format=serialization.PublicFormat.Raw,
    )).decode("utf-8")
    der_sig = signing_key.sign(session_pub.encode("utf-8"), ec.ECDSA(hashes.SHA256()))
    r, s = decode_dss_signature(der_sig)
    raw_sig = r.to_bytes(32, "big") + s.to_bytes(32, "big")

    verify_provider_session_key(
        base64.b64encode(signing_pub).decode("utf-8"),
        session_pub,
        base64.b64encode(raw_sig).decode("utf-8"),
    )
