from __future__ import annotations

import base64
import json
import os
from dataclasses import dataclass

from cryptography.exceptions import InvalidSignature
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import ec, x25519
from cryptography.hazmat.primitives.asymmetric.utils import encode_dss_signature
from cryptography.hazmat.primitives.ciphers.aead import AESGCM


@dataclass(slots=True)
class EncryptedEnvelope:
    version: int
    ephemeral_pubkey: str
    nonce: str
    ciphertext: str

    def to_json(self) -> str:
        return json.dumps(
            {
                "version": self.version,
                "ephemeral_pubkey": self.ephemeral_pubkey,
                "nonce": self.nonce,
                "ciphertext": self.ciphertext,
            }
        )


def verify_provider_session_key(
    provider_signing_pubkey: str,
    provider_session_pubkey: str,
    provider_session_signature: str,
) -> None:
    public_key = serialization.load_der_public_key(base64.b64decode(provider_signing_pubkey))
    assert isinstance(public_key, ec.EllipticCurvePublicKey)
    signature = base64.b64decode(provider_session_signature)
    if len(signature) != 64:
        raise InvalidSignature("provider session signature must be 64 raw bytes")
    der_signature = _raw_ecdsa_to_der(signature)
    public_key.verify(der_signature, provider_session_pubkey.encode("utf-8"), ec.ECDSA(hashes.SHA256()))


def encrypt_job_envelope(provider_session_pubkey: str, prompt: str, max_output_tokens: int) -> EncryptedEnvelope:
    provider_public = x25519.X25519PublicKey.from_public_bytes(base64.b64decode(provider_session_pubkey))
    ephemeral_private = x25519.X25519PrivateKey.generate()
    shared_secret = ephemeral_private.exchange(provider_public)
    digest = hashes.Hash(hashes.SHA256())
    digest.update(shared_secret)
    digest.update(b"dginf-envelope-v1")
    key = digest.finalize()
    nonce = os.urandom(12)
    aead = AESGCM(key)
    plaintext = json.dumps(
        {
            "prompt": prompt,
            "max_output_tokens": max_output_tokens,
        }
    ).encode("utf-8")
    ciphertext = aead.encrypt(nonce, plaintext, None)
    return EncryptedEnvelope(
        version=1,
        ephemeral_pubkey=base64.b64encode(
            ephemeral_private.public_key().public_bytes(
                encoding=serialization.Encoding.Raw,
                format=serialization.PublicFormat.Raw,
            )
        ).decode("utf-8"),
        nonce=base64.b64encode(nonce).decode("utf-8"),
        ciphertext=base64.b64encode(ciphertext).decode("utf-8"),
    )


def _raw_ecdsa_to_der(signature: bytes) -> bytes:
    r = int.from_bytes(signature[:32], byteorder="big")
    s = int.from_bytes(signature[32:], byteorder="big")
    return encode_dss_signature(r, s)
