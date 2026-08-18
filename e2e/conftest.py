import os
import base64
import hashlib
import secrets
import sqlite3
import time

import pytest
from dotenv import load_dotenv
from faker import Faker
from grpc_requests import Client

load_dotenv()
fake = Faker()

GRPC_ADDR = os.environ.get("GRPC_ADDR", "localhost:50051")
ACCOUNT_SERVICE = os.environ.get(
    "GRPC_ACCOUNT_SERVICE", "account.v1.AccountService")
AUTH_SERVICE = os.environ.get("GRPC_AUTH_SERVICE", "auth.v1.AuthService")
SQLITE_PATH = os.environ.get("SQLITE_PATH", "./data/app.db")


@pytest.fixture(scope="session")
def grpc_client():
    """
    Reflection-based client: same mechanism grpcui uses under the hood
    (server reflection), but scriptable. No .proto files or protoc-generated
    stubs needed, as long as the server exposes grpc reflection — which it
    must, since grpcui is already talking to it.
    """
    client = Client.get_by_endpoint(GRPC_ADDR)
    for service in (ACCOUNT_SERVICE, AUTH_SERVICE):
        assert service in client.service_names, (
            f"{service!r} not visible via reflection. "
            f"Services the server actually exposes: {client.service_names}"
        )
    return client


@pytest.fixture()
def db():
    """Fresh sqlite connection per test."""
    conn = sqlite3.connect(SQLITE_PATH)
    conn.row_factory = sqlite3.Row
    yield conn
    conn.close()


def wait_for_row(db_conn, query, params, timeout=5.0, interval=0.2):
    """
    Poll sqlite until a row shows up or timeout expires.
    Use this instead of a single SELECT right after a gRPC call —
    if account creation/exchange writes asynchronously, a single
    immediate check is flaky.
    """
    deadline = time.time() + timeout
    while time.time() < deadline:
        row = db_conn.execute(query, params).fetchone()
        if row is not None:
            return row
        time.sleep(interval)
    return None


@pytest.fixture()
def create_exchange_authenticate_flow_data():
    """
    Unique, realistic data per test run — this is the 'no hardcode' part.
    Add any other fields CreateAccount actually requires.
    """
    # 1. Generate a cryptographically secure random string for the code_verifier
    # Allowed characters: [A-Z], [a-z], [0-9], "-", ".", "_", "~"
    allowed_chars = (
        "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
    )
    verifier = "".join(secrets.choice(allowed_chars)
                       for _ in range(64))

    # 2. Hash the verifier using SHA-256
    hashed = hashlib.sha256(verifier.encode("utf-8")).digest()

    # 3. Base64URL encode the raw hash and strip padding characters (=)
    challenge = (
        base64.urlsafe_b64encode(hashed).decode("utf-8").rstrip("=")
    )
    return {
        "username": fake.name(),
        "email": fake.unique.email(),
        "password": fake.password(length=16),
        "challenge": challenge,
        "verifier": verifier,
    }
