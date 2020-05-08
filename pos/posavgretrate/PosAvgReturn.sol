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



    function addTest() public view returns(uint256 retx, uint256 rety,bool success) {
       uint256 x1 = 0x69088a1c79a78b5e66859a5e6594d70c8f12a1ff882d84a05ffdbbcff5a4abcb;
       uint256 y1 = 0x5d4c67c05b0a693fb72b47abf7e0d6381fc722ca45c8bb076e6cb4f9f0912906;

       uint256 x2 = 0xfb4a50e7008341df6390ad3dcd758b1498959bf18369edc335435367088910c6;
       uint256 y2 = 0xe55f58908701c932768c2fd16932f694acd30e21a5f2a4f6242b5f0567696240;

       return add(x1,y1,x2,y2);
    }

    function add(uint256 x1, uint256 y1, uint256 x2,uint256 y2)  public view returns(uint256 retx, uint256 rety,bool success) {

       bytes32 functionSelector =0xe022d77c00000000000000000000000000000000000000000000000000000000;
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

     function mulGTest() public view returns(uint256 retx, uint256 rety,bool success){
         uint256 scalar = 2;
         return mulG(scalar);
     }

    function mulG(uint256 scalar)   public view returns(uint256 x, uint256 y,bool success) {
        bytes32 functionSelector = 0xbb734c4e00000000000000000000000000000000000000000000000000000000;//keccak256("mulG(uint256)");
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


    function hexStr2bytes(string data)returns (bytes){
        uint _ascii_0 = 48;
        uint _ascii_A = 65;
        uint _ascii_a = 97;

        bytes memory a = bytes(data);
        uint[] memory b = new uint[](a.length);

        for (uint i = 0; i < a.length; i++) {
            uint _a = uint(a[i]);

            if (_a > 96) {
                b[i] = _a - 97 + 10;
            }
            else if (_a > 66) {
                b[i] = _a - 65 + 10;
            }
            else {
                b[i] = _a - 48;
            }
        }

        bytes memory c = new bytes(b.length / 2);
        for (uint _i = 0; _i < b.length; _i += 2) {
            c[_i / 2] = byte(b[_i] * 16 + b[_i + 1]);
        }

        return c;
    }


    string pk = "042bda949acb1f1d5e6a2952c928a0524ee088e79bb71be990274ad0d3884230544b0f95d167eef4f76962a5cf569dabc018d025d7494986f7f0b11af7f0bdcbf4";
    string poly = "c2d0052ee30a386ff1e08c948a4a7a654345d17e9a8fe1b6b4d2cf8c305dd16fa94b83e74e4f17d4a96044d51e038b86fa71a8220e328967693f183545711174c8f5fd08e531047a06e3c1093a22e0594c49245a8f5f5993a537a049d44ac03ca27b317cf91e779e0eba32d73e6958c38754c385020c06ea239814c6dd649db3821274ce0a8e505c3589da13437880d94aaf6dd53eeb55cd1d1e794f070a00a43ac85d55ae00d78eb7dfa9b5db63d16e559bc40e96c012799dc88c8cda2031d503c33f5e39216a58ca35afdb767b60a3248044ec2dd8374e05beb5cc7636c2e3eff123930b86a8534bb6c28943e080a5332844f1947208363f0ba5e15ea24e6e890b7356e82a61e51ac7dc6718c38f9fa2ba22d9ec01623e87db59c95612230128afe265149efa3d957e18e862aa6b26d68cefb0e7d1e6665df08aa47176f09007efa5e4bc433e9a6b4ca6d44356d2b290005fb3c3101dc2efaef4e87f8d9523a44a12418ddf1ac0caa9d1173381461d247793efb1619645e17f55c4e2cc732d17982cbbf093b009fd263ca6845814650c2b3cbaabca87761fa8cbc260993d911328f5a422e6823971effd422c09590029a2ad4643e402648621b21d7b0f205c136741660d40fb4ba37fa093e8ba806996a3d4c6d6170bc5043cd7d3f7f38aac75d57e6e0e75529d8478e43cb4f44f01c4f2c7ce9c29bd6a2dda2765d7bf407f44679a4c06d14b961d3f549850ac882cba2720b0ce4bde2f0a298f2801802fb0747cd4440a551ea600639f87b4bd5257aad2a96136502a584667aaafcd39d865";


    function calPolyCommitTest()   public view returns(uint256 sx, uint256 sy,bool success) {

         bytes memory bpk = hexStr2bytes(pk);
         bytes memory bpoly = hexStr2bytes(poly);

         return calPolyCommit(bpoly,bpk);

    }


    function calPolyCommit(bytes polyCommit, bytes pk)   public view returns(uint256 sx, uint256 sy,bool success) {

       bytes32 functionSelector = 0xf9d9c3ff00000000000000000000000000000000000000000000000000000000;//keccak256("calPolyCommit(bytes,uint256)");
       address to = PRECOMPILE_CONTRACT_ADDR;

       uint segcnt = (polyCommit.length + pk.length)/32 + 1;

       bytes memory wholeData = new bytes(segcnt*32);

       uint i = 0;
       for(i=0;i<polyCommit.length;i++) {
           wholeData[i] = polyCommit[i];
       }

       uint j =0;
       for(j=0;j <pk.length;j++) {
            wholeData[i++] = pk[j];
       }

       for(;i<segcnt*32;i++) {
            wholeData[i] = bytes1(pk.length);
       }
      // uint loopCnt = 0;

       assembly {
            let loopCnt := 0
            let freePtr := mload(0x40)
            mstore(freePtr, functionSelector)

            loop:
                jumpi(loopend, eq(loopCnt,segcnt))
                mstore(add(freePtr,add(4,mul(loopCnt,32))), mload(add(wholeData,mul(add(loopCnt,1),32))))
                loopCnt := add(loopCnt, 1)
                jump(loop)
            loopend:

            success := staticcall(gas,to, freePtr,add(mul(segcnt,32),4), freePtr, 64)

            sx := mload(freePtr)
            sy := mload(add(freePtr, 32))
        }
    }



    function enc(uint256 r, uint256 M, bytes K)   public view returns (bytes c,bool success) {
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