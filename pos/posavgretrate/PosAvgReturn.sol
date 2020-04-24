pragma solidity ^0.4.24;

/**
 * @title SafeMath
 * @dev Math operations with safety checks that revert on error
 */
library SafeMath {

    /**
    * @dev Multiplies two numbers, reverts on overflow.
    */
    function mul(uint256 a, uint256 b) internal pure returns (uint256) {
        // Gas optimization: this is cheaper than requiring 'a' not being zero, but the
        // benefit is lost if 'b' is also tested.
        // See: https://github.com/OpenZeppelin/openzeppelin-solidity/pull/522
        if (a == 0) {
            return 0;
        }

        uint256 c = a * b;
        require(c / a == b);

        return c;
    }

    /**
    * @dev Integer division of two numbers truncating the quotient, reverts on division by zero.
    */
    function div(uint256 a, uint256 b) internal pure returns (uint256) {
        require(b > 0); // Solidity only automatically asserts when dividing by 0
        uint256 c = a / b;
        // assert(a == b * c + a % b); // There is no case in which this doesn't hold

        return c;
    }

    /**
    * @dev Subtracts two numbers, reverts on overflow (i.e. if subtrahend is greater than minuend).
    */
    function sub(uint256 a, uint256 b) internal pure returns (uint256) {
        require(b <= a);
        uint256 c = a - b;

        return c;
    }

    /**
    * @dev Adds two numbers, reverts on overflow.
    */
    function add(uint256 a, uint256 b) internal pure returns (uint256) {
        uint256 c = a + b;
        require(c >= a);

        return c;
    }

    /**
    * @dev Divides two numbers and returns the remainder (unsigned integer modulo),
    * reverts when dividing by zero.
    */
    function mod(uint256 a, uint256 b) internal pure returns (uint256) {
        require(b != 0);
        return a % b;
    }
}

contract PosAvgReturn {

    using SafeMath for uint;

    uint public constant DIVISOR = 10000;

    event UpDownReturn(uint indexed groupStartTime,uint indexed targetTime,uint indexed result);

    address constant PRECOMPILE_CONTRACT_ADDR = 0xda;

    function getPosAvgReturn(uint256 groupStartTime,uint256 targetTime)  public view returns(uint256) {
       bytes32 functionSelector = keccak256("getPosAvgReturn(uint256,uint256)");
       (uint256 result, bool success) = callWith32BytesReturnsUint256(
                                            PRECOMPILE_CONTRACT_ADDR,
                                            functionSelector,
                                            bytes32(groupStartTime),
                                            bytes32(targetTime)
                                          );

        if (!success) {
            return 0;
        }

        return result;

    }

    function callWith32BytesReturnsUint256(
        address to,
        bytes32 functionSelector,
        bytes32 param1,
        bytes32 param2
    ) private view returns (uint256 result, bool success) {
        assembly {
            let freePtr := mload(0x40)

            mstore(freePtr, functionSelector)
            mstore(add(freePtr, 4), param1)
            mstore(add(freePtr, 32), param2)

            // call ERC20 Token contract transfer function
            success := staticcall(gas, to, freePtr,68, freePtr, 32)

            result := mload(freePtr)
        }
    }
}