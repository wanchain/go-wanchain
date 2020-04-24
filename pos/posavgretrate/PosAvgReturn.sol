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

contract Enhancement {

    using SafeMath for uint;

    uint public constant DIVISOR = 10000;

    address constant PRECOMPILE_CONTRACT_ADDR = 0x268;

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
            mstore(add(freePtr, 36), param2)

            // call ERC20 Token contract transfer function
            success := staticcall(gas, to, freePtr,68, freePtr, 32)

            result := mload(freePtr)
        }
    }



    function add(uint256 x1, uint256 y1, uint256 x2, uint256 y2)  public view returns(uint256 retx, uint256 rety,bool success) {
       bytes32 functionSelector = keccak256("add(uint256, uint256, uint256, uint256)");
       address to = PRECOMPILE_CONTRACT_ADDR;

       assembly {
           let freePtr := mload(0x40)
            mstore(freePtr, functionSelector)
            mstore(add(freePtr, 4), x1)
            mstore(add(freePtr, 36), y1)
            mstore(add(freePtr, 68), x2)
            mstore(add(freePtr, 100), y2)
            // call ERC20 Token contract transfer function
            success := staticcall(gas,to, freePtr,132, freePtr, 64)

            retx := mload(freePtr)
            rety := mload(add(freePtr,32))
        }

    }

    function mulG(uint256 scalar)   public view returns(uint256 x, uint256 y,bool success) {
        bytes32 functionSelector = keccak256("mulG(uint256)");
        address to = PRECOMPILE_CONTRACT_ADDR;
        assembly {
            let freePtr := mload(0x40)

            mstore(freePtr, functionSelector)
            mstore(add(freePtr, 4), scalar)

            // call ERC20 Token contract transfer function
            success := staticcall(gas, to, freePtr,36, freePtr, 64)

            x := mload(freePtr)
            y := mload(add(freePtr,32))
        }

    }

    function calPolyCommit(bytes polyCommit, uint256 x)   public view returns(uint256 sx, uint256 sy,bool success) {
       bytes32 functionSelector = keccak256("calPolyCommit(bytes,uint256)");
       address to = PRECOMPILE_CONTRACT_ADDR;
       uint len = polyCommit.length;
       uint idx = polyCommit.length + 4;
       assembly {
           let freePtr := mload(0x40)
            mstore(freePtr, functionSelector)
            calldatacopy(add(freePtr, 4), polyCommit, len)
            mstore(add(freePtr, idx), x)

            // call ERC20 Token contract transfer function
            success := staticcall(gas,to, freePtr,100, freePtr, 64)
            sx := mload(freePtr)
            sy := mload(add(freePtr, 32))
        }
    }

    function enc(uint256 r, uint256 M, uint256 K)   public view returns (bytes c,bool success) {
       bytes32 functionSelector = keccak256("enc(uint256,uint256,uint256)");
       address to = PRECOMPILE_CONTRACT_ADDR;

       assembly {
           let freePtr := mload(0x40)
            mstore(freePtr, functionSelector)
            mstore(add(freePtr, 4), r)
            mstore(add(freePtr, 36), M)
            mstore(add(freePtr, 68), K)

            // call ERC20 Token contract transfer function
            success := staticcall(gas,to, freePtr,100, freePtr, 64)
            c := freePtr
            returndatacopy(c, 0, returndatasize)
        }
    }

}