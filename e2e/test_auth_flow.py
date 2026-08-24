"""
E2E: CreateAccount -> AuthenticateAccount -> ExchangeAuthCode -> verify in sqlite.

This is the scripted equivalent of the manual grpcui flow:
  1. CreateAccount
  2. AuthenticateAccount with the same params
  3. grab auth code from the response (no copy-paste)
  4. ExchangeAuthCode with the code + credentials
  5. assert the record exists in sqlite
"""

from conftest import ACCOUNT_SERVICE, AUTH_SERVICE, wait_for_row


def test_create_authenticate_exchange_flow(grpc_client, db, create_exchange_authenticate_flow_data):
    # 1. CreateAccount
    _ = grpc_client.request(ACCOUNT_SERVICE, "CreateAccount", {
        "username": create_exchange_authenticate_flow_data["username"],
        "email": create_exchange_authenticate_flow_data["email"],
        "password": create_exchange_authenticate_flow_data["password"],
    })

    # 2. AuthenticateAccount — same credentials as above, nothing re-typed
    auth_resp = grpc_client.request(AUTH_SERVICE, "AuthenticateAccount", {
        "email": create_exchange_authenticate_flow_data["email"],
        "password": create_exchange_authenticate_flow_data["password"],
        "challenge": create_exchange_authenticate_flow_data["challenge"],
    })

    # Adjust the field name below to match your actual response schema
    auth_code = auth_resp["auth_code"]
    assert auth_code, "AuthenticateAccount did not return an auth code"

    # 3. ExchangeAuthCode — code taken straight from the previous response
    exchange_resp = grpc_client.request(AUTH_SERVICE, "ExchangeAuthCode", {
        "code": auth_code,
        "verifier": create_exchange_authenticate_flow_data["verifier"],
    })
    assert exchange_resp, "ExchangeAuthCode returned an empty response"
    assert exchange_resp["refresh_token"], "ExchangeAuthCode returned no refresh token"
    assert exchange_resp["access_token"], "ExchangeAuthCode returned no access token"

    # 4. Verify the record landed in sqlite — adjust table/column names
    row = wait_for_row(
        db,
        "SELECT * FROM refresh_tokens WHERE token_hash = ?",
        (exchange_resp["refresh_token"],),
    )
    assert row is not None, "No matching row appeared in sqlite after exchange"

    refresh_resp = grpc_client.request(AUTH_SERVICE, "RefreshAccessToken", {
        "refresh_token": exchange_resp["refresh_token"],
    })
    assert refresh_resp, "RefreshAccessToken returned an empty response"

    row = wait_for_row(
        db,
        "SELECT * FROM refresh_tokens WHERE token_hash = ?",
        (refresh_resp["refresh_token"],),
    )
    assert row is not None, "No matching row appeared in sqlite after exchange"

    row = wait_for_row(
        db,
        "SELECT * FROM refresh_tokens WHERE token_hash = ?",
        (exchange_resp["refresh_token"],),
    )
    assert row["revoked_at"] is not None, "Old refresh token is not revoked"
