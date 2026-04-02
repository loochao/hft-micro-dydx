#!/usr/bin/env python3
"""
dYdX v4 Order Bridge for Go HFT System
Phase 1: Uses dydx-v4-client Python package to place/cancel orders
via Cosmos SDK transactions.

Usage:
  python3 place_order.py --action place --mnemonic "word1 word2 ..." \
    --market BTC-USD --side BUY --type LIMIT --size 0.001 --price 50000 \
    --time-in-force GTT --client-id "123456" --subaccount-number 0

  python3 place_order.py --action cancel --mnemonic "word1 word2 ..." \
    --market BTC-USD --client-id "123456" --subaccount-number 0

Output: JSON {"success": true/false, "orderId": "...", "error": "..."}
"""

import argparse
import json
import sys
import asyncio
import os

def main():
    parser = argparse.ArgumentParser(description="dYdX v4 order bridge")
    parser.add_argument("--action", required=True, choices=["place", "cancel"])
    parser.add_argument("--mnemonic", required=True)
    parser.add_argument("--market", required=True)
    parser.add_argument("--side", default="BUY")
    parser.add_argument("--type", dest="order_type", default="LIMIT")
    parser.add_argument("--size", type=float, default=0.0)
    parser.add_argument("--price", type=float, default=0.0)
    parser.add_argument("--time-in-force", default="GTT")
    parser.add_argument("--client-id", default="")
    parser.add_argument("--subaccount-number", type=int, default=0)
    parser.add_argument("--post-only", action="store_true")
    parser.add_argument("--reduce-only", action="store_true")
    parser.add_argument("--proxy", default="")
    args = parser.parse_args()

    try:
        result = asyncio.run(execute_order(args))
        print(json.dumps(result))
    except Exception as e:
        print(json.dumps({"success": False, "orderId": "", "error": str(e)}))
        sys.exit(0)  # Exit 0 so Go can parse the JSON error


async def execute_order(args):
    try:
        from dydx_v4_client import NodeClient, Wallet  # type: ignore
        from dydx_v4_client.node.market import Market  # type: ignore
        from dydx_v4_client.indexer.rest.constants import MAINNET_API_URL  # type: ignore
    except ImportError:
        # Fallback: try the v4-proto based client
        try:
            from v4_client_py import IndexerClient, ValidatorClient  # type: ignore
            from v4_client_py.chain.aerial.wallet import LocalWallet  # type: ignore
            from v4_client_py.clients.constants import Network  # type: ignore
            return await execute_order_v4_client_py(args)
        except ImportError:
            return {
                "success": False,
                "orderId": "",
                "error": "Neither dydx_v4_client nor v4_client_py is installed. "
                         "Install with: pip install v4-client-py"
            }

    # Use dydx_v4_client path
    return await execute_order_dydx_v4_client(args)


async def execute_order_v4_client_py(args):
    """Order execution using v4-client-py package"""
    from v4_client_py import IndexerClient, ValidatorClient  # type: ignore
    from v4_client_py.chain.aerial.wallet import LocalWallet  # type: ignore
    from v4_client_py.clients.constants import Network  # type: ignore
    from v4_client_py.clients.helpers.chain_helpers import (  # type: ignore
        OrderSide, OrderType, OrderTimeInForce, OrderExecution
    )

    network = Network.mainnet()
    client = ValidatorClient(network.validator_config)
    wallet = LocalWallet.from_mnemonic(args.mnemonic, "dydx")

    if args.action == "place":
        # Map side
        side = OrderSide.BUY if args.side == "BUY" else OrderSide.SELL

        # Map time in force
        if args.time_in_force == "IOC":
            time_in_force = OrderTimeInForce.IOC
        elif args.time_in_force == "FOK":
            time_in_force = OrderTimeInForce.FOK
        else:
            time_in_force = OrderTimeInForce.GTT

        # Map order type
        if args.order_type == "MARKET":
            order_type = OrderType.MARKET
        else:
            order_type = OrderType.LIMIT

        # For short-term orders, use good_til_block
        # For long-term orders, use good_til_block_time
        import time
        good_til_block_time = int(time.time()) + 120  # 2 minutes

        # Get current block for short-term orders
        try:
            current_block = client.get_current_block()
            good_til_block = current_block + 20  # ~20 blocks ahead
        except Exception:
            good_til_block = 0

        # Place the order
        try:
            tx = client.post.place_order(
                wallet,
                args.subaccount_number,
                args.client_id or str(int(time.time() * 1000)),
                args.market,
                order_type,
                side,
                args.price,
                args.size,
                good_til_block if good_til_block > 0 else good_til_block_time,
                time_in_force,
                reduce_only=args.reduce_only,
            )
            return {
                "success": True,
                "orderId": args.client_id,
                "error": ""
            }
        except Exception as e:
            return {
                "success": False,
                "orderId": "",
                "error": f"place_order failed: {str(e)}"
            }

    elif args.action == "cancel":
        try:
            # Get current block for cancel
            current_block = client.get_current_block()
            good_til_block = current_block + 20

            tx = client.post.cancel_order(
                wallet,
                args.subaccount_number,
                args.client_id,
                args.market,
                0,  # order_flags
                good_til_block,
            )
            return {
                "success": True,
                "orderId": args.client_id,
                "error": ""
            }
        except Exception as e:
            return {
                "success": False,
                "orderId": "",
                "error": f"cancel_order failed: {str(e)}"
            }


async def execute_order_dydx_v4_client(args):
    """Order execution using dydx_v4_client package"""
    return {
        "success": False,
        "orderId": "",
        "error": "dydx_v4_client path not yet implemented, use v4-client-py"
    }


if __name__ == "__main__":
    main()
