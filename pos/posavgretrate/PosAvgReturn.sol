pragma solidity ^0.4.24;

import "../SafeMath.sol";

contract PosAvgReturn {

    using SafeMath for uint;

    uint public constant DIVISOR = 10000;

    event UpDownReturn(uint indexed groupStartTime,uint indexed targetTime,uint indexed result);

    address constant PRECOMPILE_CONTRACT_ADDR = 0xda;
    bytes32 constant GET_POS_AVG_RET_SELECTOR = 0x8c114a5100000000000000000000000000000000000000000000000000000000;

    function getPosAvgReturn(uint256 groupStartTime,uint256 targetTime)  public view returns(uint256) {

        (uint256 result, bool success) = callWith32BytesReturnsUint256(
                                            PRECOMPILE_CONTRACT_ADDR,
                                            GET_POS_AVG_RET_SELECTOR,
                                            bytes32(groupStartTime),
                                            bytes32(targetTime)
                                          );

        if (!success) {
            revert("ASSEMBLY_CALL_GET_POS_AVG_RET_FAILED");
        }

        emit UpDownReturn(groupStartTime,targetTime,result);

        return result;
    }

   function callWith32BytesReturnsUint256(
        address to,
        bytes32 functionSelector,
        bytes32 param1,
        bytes32 param2
    )
        private
        view
        returns (uint256 result, bool success)
    {
        assembly {
            let freePtr := mload(0x40)
            let tmp1 := mload(freePtr)
            let tmp2 := mload(add(freePtr, 4))

            mstore(freePtr, functionSelector)
            mstore(add(freePtr, 4), param1)
            mstore(add(freePtr, 36), param2)

            // call ERC20 Token contract transfer function
            success := staticcall(
                gas,           // Forward all gas
                to,            // Interest Model Address
                freePtr,       // Pointer to start of calldata
                68,            // Length of calldata，4+32+32
                freePtr,       // Overwrite calldata with output
                32             // Expecting uint256 output
            )

            result := mload(freePtr)

            mstore(freePtr, tmp1)
            mstore(add(freePtr, 4), tmp2)
        }
    }
}