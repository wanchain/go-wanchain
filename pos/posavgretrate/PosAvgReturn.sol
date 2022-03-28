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
    
    function getEpochId(uint256 blockTime) public view returns (uint256) {
        bytes32 functionSelector = keccak256("getEpochId(uint256)");

        (uint256 result, bool success) = callWith32BytesReturnsUint256(
            0x262,
            functionSelector,
            bytes32(blockTime)
        );

        require(success, "ASSEMBLY_CALL getEpochId failed");

        return result;
    }  
  
    function callWith32BytesReturnsUint256(
        address to,
        bytes32 functionSelector,
        bytes32 param1
    ) private view returns (uint256 result, bool success) {
        assembly {
            let freePtr := mload(0x40)

            mstore(freePtr, functionSelector)
            mstore(add(freePtr, 4), param1)

            // call ERC20 Token contract transfer function
            success := staticcall(gas, to, freePtr, 36, freePtr, 32)

            result := mload(freePtr)
        }
    }  
  
    
    function getPosAvgReturn(uint256 groupStartTime,uint256 curTime)  public view returns(uint256 result,bool success) {
      // bytes32 functionSelector = keccak256("getPosAvgReturn(uint256,uint256)");
       bytes32 functionSelector = 0x8c114a5100000000000000000000000000000000000000000000000000000000;
       address to = PRECOMPILE_CONTRACT_ADDR;

       assembly {
            let freePtr := mload(0x40)
            mstore(freePtr, functionSelector)
            mstore(add(freePtr, 4), groupStartTime)
            mstore(add(freePtr, 36), curTime)

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
    string poly = "0477947c2048cefbeb637ca46d98a1992c8f0a832e288be5adb36bce9ffb7965deef0024de93f1c30255a6b7deec2ba09d14f0c2f457416098b8266bb16a67e52004e84e2ab12f974cea11c948d276ce38b75638907f3259e8c60db07cf80b492d7da5a4c6e915ab16ba695a9825e6e4441cc843016100534fbce9a7d947d290afc904d665dd602ca1bc43245843dd4721dc7e4509b89c0b94e4744366c4ec491e9aad6efde662ab34bc836724db7f8613ff9131986fc21338e0f2352134b7f915f3d80425e027d24a8c65c0264ae8afbc4218cdd72266f8f245017b8725ef730ad4e80884dd77fbac60297ff6cf5cf6cb130b03b4551605cb5fc85f23ad98a9c6ea24d204367763779f7857ff97a304042885516f70e215ba57852d2763692ea8c6be93a7af3551a2014f7d2a1174335ce69808c57b8dc3c8b2f4ae948696052d8b81034304f6c5c039d2dc4d70aad4baefec8e31a5cc9ebd628cda32da8ed770189cf0dee3d5d5688618ff76e46bd3d40b1aa68b122c5c73af09060c065900790c68ee535304eff4a83c31442c94afd04414d7d4a41ecc20dfd6c587b94fd6a0398555c5dacf350411dab79965e9ef184b443b711b666aa290cfb0e2c263a317be9d0d3ec79a049eb4a277716d47fb868daab644eb66f0fff79a931b483af19a11fb2d097d59c09e73d02d7de04f099f463f10a368334e5b94a618eb6dfd80cfa29f6d9c5832e4047f33a451cb89f81d03823b73bbcc3e3efcaddc015c5e2907d2d4a9535eb6ecf23790c8451554319cec0848b1043281fde3d656e4d89f4041718221ad91cbd71a04e6b755737ccb1afcf5a839869a6d6dab529d263796a06e839190b25a45b31c8696659dade33df0be779a2d3aa987810bcf85d45a7e4d905c3ecf0b977a5dfc9f044c9c5be87bd1f4b334b4a34eac2fac1fb45a248eb071a077fb65e725670fa2367a9ffdb79233769859d44511f01f17a8eb3ae5092c739f2f37d07d656c440cd4043c188a61cdf98bc160935134a039acf3bf1a76d5389841fe93e93317fae34bc15d26c76d926650944c1d8c696212d48691540b04a362ff9e710f8fba967fb58004e919ca4d9a9f59b925579c17fd27fddbf144259a64562051cd93f1672729c3cb24ef17632d7538aa0f49c44b591f26685d3e0edba529e8f868f091839802c037043680e14d808cb3d9f34243204b16f6cdaf172253100526b3a774bc5cb1cbd70d2f9f5f52793b5aeb8b2e22861be26f71ee762aed65b983910fcfe6cab00d4f1704e03eee5f2f37368d687350ee6088d5255263c145ac7c65d630a2d3d7f81452a7d474e5f92e76f0fafddec74e4b0cc65499a34965e6485e3474166a21d6262cbc0444ca736fcd0476b316701d4c636f4abe69bca60e9f66f80293d821fdf3549d604c45dabc802c75c68ff9de8dff63e946d62a44c99c108558addd4568f63cdc66047021ed3d4f2d75ec7dbbdb4fffd429f9784cd4781481b6bb03f80673190751f0cb5f4d690ded3c1cecd9181fab90ed34bec67c1af519caa36e8c24bdd6430901";


    function calPolyCommitTest()   public view returns(uint256 sx, uint256 sy,bool success) {
        
         bytes memory bpk = hexStr2bytes(pk);
         bytes memory bpoly = hexStr2bytes(poly);
         
         return calPolyCommit(bpoly,bpk);
        
    }
    
    
    function calPolyCommit(bytes polyCommit, bytes pk)   public view returns(uint256 sx, uint256 sy,bool success) {
        
       bytes32 functionSelector = 0xf9d9c3ff00000000000000000000000000000000000000000000000000000000;//keccak256("calPolyCommit(bytes,uint256)");
       address to = PRECOMPILE_CONTRACT_ADDR;
      
       require((polyCommit.length + pk.length)%65 == 0);
       
       uint polyCommitCnt = polyCommit.length/65;
       uint total = (polyCommitCnt + 1)*2;
       
       assembly {
            let freePtr := mload(0x40)
            mstore(freePtr, functionSelector)
            mstore(add(freePtr,4), mload(add(polyCommit,33)))
            mstore(add(freePtr,36), mload(add(polyCommit,65)))
            let loopCnt := 1
            loop:
                jumpi(loopend, eq(loopCnt,polyCommitCnt))
                mstore(add(freePtr,add(4,mul(loopCnt,64))),         mload(add(add(add(polyCommit,32),mul(loopCnt,65)),1)))
                mstore(add(freePtr,add(4,add(mul(loopCnt,64),32))), mload(add(add(add(add(polyCommit,32),mul(loopCnt,65)),1),32)))
                loopCnt := add(loopCnt, 1)
                jump(loop)
            loopend:
            
            mstore(add(freePtr,    add(4,mul(loopCnt,64))),     mload(add(pk,33)))
            mstore(add(freePtr,add(add(4,mul(loopCnt,64)),32)), mload(add(pk,65)))

            success := staticcall(gas,to, freePtr,add(mul(total,32),4), freePtr, 64)
            
            sx := mload(freePtr)
            sy := mload(add(freePtr, 32))
        }
        
    
    }


// msg = 334dd1fdf10ae8dc29e1d2e46309a8700b95cc103bad30c246901be4a4f9e130	

// Private key-1:[8cdf775872f31d7ddcc18000edcbede9463b6c65c4d288af776075addb25a7c7]
// Public key-1:[041fc83598dbd36792d8246c7651631535953c741bf93ba32f1dc2960e92fc2f276fdfb930c4a84b39bc84b0f149323e5a6b10092acaf31f31b9f8a0946f6643d0]

// rbpriv=53b7bccfd434a359b56f5bac592935802c3e30f6e69e30e0bc6bb4ac1108f1b7
// rbpub=03a804abfeff33e966dabe0bbdb7ab6bca5c54c96b5000ae41391d9f30e9aeb63a

// rbpubuncom=04a804abfeff33e966dabe0bbdb7ab6bca5c54c96b5000ae41391d9f30e9aeb63a05f49fb1adbbff4e7390ce29381db7c8c14c442d4a773891a9326c6db2c6792b
// sharedKey=123143d6ffd6768895c859c05353517eb4631c1060a6433ff346e7a400b2aa5a
// derivedKey=5b90ee0ee834050fc91e02269fabf2759d08b52353a189e70dac5ff25d9fe02692cf212a0ffd3fab2ea893cc6e056e63a3dffad3734c78d419747dcea0bff4bc

// encKey=5b90ee0ee834050fc91e02269fabf275
// macKey=9d08b52353a189e70dac5ff25d9fe026
// hashmacKey=469910462f8d192217f7b0aeb960dbf2769425524165ac8c03007e0432c5f591
// iv=c5fd3f6b773f094d146a255946114822

// encryptedMsg=b382f2e2b6bd7a310fa6961efa3d84c60028aa9224148c221d3c8a697bec289fd09601985302264c1033de1c282e456f

// mac=b382f2e2b6bd7a310fa6961efa3d84c60028aa9224148c221d3c8a697bec289fd09601985302264c1033de1c282e456f

// eccEncWhole=04a804abfeff33e966dabe0bbdb7ab6bca5c54c96b5000ae41391d9f30e9aeb63a05f49fb1adbbff4e7390ce29381db7c8c14c442d4a773891a9326c6db2c6792bc5fd3f6b773f094d146a255946114822b382f2e2b6bd7a310fa6961efa3d84c60028aa9224148c221d3c8a697bec289fd09601985302264c1033de1c282e456f332f808ae60eb6cb75123153d6e7767627bce7f0d752f866116727185e2d7d7b

// 04a804abfeff33e966dabe0bbdb7ab6bca5c54c96b5000ae41391d9f30e9aeb63a05f49fb1adbbff4e7390ce29381db7c8c14c442d4a773891a9326c6db2c6792bc5fd3f6b773f094d146a255946114822b382f2e2b6bd7a310fa6961efa3d84c60028aa9224148c221d3c8a697bec289fd09601985302264c1033de1c282e456f332f808ae60eb6cb75123153d6e7767627bce7f0d752f866116727185e2d7d7b

// 0x04a804abfeff33e966dabe0bbdb7ab6bca5c54c96b5000ae41391d9f30e9aeb63a05f49fb1adbbff4e7390ce29381db7c8c14c442d4a773891a9326c6db2c6792bc5fd3f6b773f094d146a255946114822b382f2e2b6bd7a310fa6961efa3d84c60028aa9224148c221d3c8a697bec289fd09601985302264c1033de1c282e456f332f808ae60eb6cb75123153d6e7767627bce7f0d752f866116727185e2d7d


 function encTest()   public view returns (bytes c,bool success) {

     bytes memory bpk = hexStr2bytes("041fc83598dbd36792d8246c7651631535953c741bf93ba32f1dc2960e92fc2f276fdfb930c4a84b39bc84b0f149323e5a6b10092acaf31f31b9f8a0946f6643d0");

     bytes32 iv = 0xc5fd3f6b773f094d146a255946114822;
     bytes32 rbpri = 0x53b7bccfd434a359b56f5bac592935802c3e30f6e69e30e0bc6bb4ac1108f1b7;

     uint256 msg = 0x334dd1fdf10ae8dc29e1d2e46309a8700b95cc103bad30c246901be4a4f9e130;

     return enc(rbpri,iv,msg,bpk);

  }

 function enc(bytes32 rbpri,bytes32 iv,uint256 mes, bytes pub)   public view returns (bytes,bool success) {
       bytes32 functionSelector = 0xa1ecea4b00000000000000000000000000000000000000000000000000000000;
       address to = PRECOMPILE_CONTRACT_ADDR;
       bytes memory cc = new bytes(6*32);
       assembly {
           let freePtr := mload(0x40)
            mstore(freePtr, functionSelector)
            mstore(add(freePtr, 4), rbpri)
            mstore(add(freePtr, 36), iv)
            mstore(add(freePtr, 68), mes)
            mstore(add(freePtr, 100), mload(add(pub,33)))
            mstore(add(freePtr, 132), mload(add(pub,65)))
            
            // call ERC20 Token contract transfer function
            success := staticcall(gas,to, freePtr,164, freePtr,1024)
            
            let loopCnt := 0
            loop:
                jumpi(loopend, eq(loopCnt,6))
                mstore(add(cc,mul(loopCnt,32)),mload(add(freePtr,mul(loopCnt,32))))
                loopCnt := add(loopCnt, 1)
                jump(loop)
            loopend:   
        }
        
        return (cc,success);
    }
    
    string senderPk="04d9482a01dd8bb0fb997561e734823d6cf341557ab117b7f0de72530c5e2f0913ef74ac187589ed90a2b9b69f736af4b9f87c68ae34c550a60f4499e2559cbfa5";
    
    bytes32 r = 0xba1d75823c0f4c07be3e07723e54c3d503829d3c9d0599a78426ac4995096a17;
    bytes32 s = 0x9a3b16eac39592d14e53b030e0275d087b9e6b38dc9d47a7383df40b4c7aec90;
    bytes32 hash = 0xb536ad7724251502d75380d774ecb5c015fd8a191dd6ceb05abf677e281b81e1;
    
    function checkSigTest () public view returns(bool) {
        bytes memory pk =hexStr2bytes(senderPk);
        return checkSig(hash,r,s,pk);
        
    }
    
    function checkSig (bytes32 hash, bytes32 r, bytes32 s, bytes pk) public view returns(bool) {
       bytes32 functionSelector = 0x861731d500000000000000000000000000000000000000000000000000000000;
       address to = PRECOMPILE_CONTRACT_ADDR;
       uint256 result;
       bool success;
       assembly {
            let freePtr := mload(0x40)

            mstore(freePtr, functionSelector)
            mstore(add(freePtr, 4), hash)
            mstore(add(freePtr, 36), r)
            mstore(add(freePtr, 68), s)
            mstore(add(freePtr, 100), mload(add(pk,33)))
            mstore(add(freePtr, 132), mload(add(pk,65)))
            
            // call ERC20 Token contract transfer function
            success := staticcall(gas, to, freePtr,164, freePtr, 32)

            result := mload(freePtr)
        }
        
        if (success) {
            return result == 1;
        } else {
            return false;
        }
    }
    
    
     function testGetHardCap ()  public view returns(uint256,bool) {
         return getHardCap(10000,10000,now - 3600 * 24);
     }
    
    
    function getHardCap (uint256 crossChainCoefficient,uint256 chainTypeCoefficient,uint256 time) public view returns(uint256,bool) {
       bytes32 functionSelector = 0xfa7c2faf00000000000000000000000000000000000000000000000000000000;
       address to = PRECOMPILE_CONTRACT_ADDR;
       uint256 posReturn;
       bool    success;
       assembly {
            let freePtr := mload(0x40)
            mstore(freePtr, functionSelector)
            mstore(add(freePtr, 4), time)
            success := staticcall(gas, to, freePtr,36, freePtr, 32)
            posReturn := mload(freePtr)
        }
        
        
        uint256 res = posReturn.mul(crossChainCoefficient).mul(chainTypeCoefficient).div(DIVISOR*DIVISOR);
        
        return (res,success);
       
    }
    

     function getMinIncentive1 ()  public view returns(uint256,uint256) {
         return (getMinIncentive(100000 ether,now - 86400 * 4,10000,10000),0);
     }


     function getMinIncentive2 ()  public view returns(uint256,uint256) {
         return (getMinIncentive(10000000 ether,now - 86400 * 4,10000,10000),0);
     }
    
    function getMinIncentive (uint256 smgDeposit,uint256 smgStartTime,uint256 crossChainCoefficient,uint256 chainTypeCoefficient) public view returns(uint256) {
        uint256 p1;
        bool    success;
        
        (p1,success) = getPosAvgReturn(smgStartTime,now);
        if(!success) {
            return 0;
        }
        uint256 p1Return = smgDeposit.mul(p1).div(DIVISOR);
        
        uint256 hardcap;
        (hardcap,success) = getHardCap(crossChainCoefficient,chainTypeCoefficient,now);
        if(!success) {
            return 0;
        }
        
        uint256 hardcapReturn = hardcap.mul(1 ether).div(DIVISOR);
        
        return hardcapReturn<=p1Return?hardcapReturn:p1Return;
    }
    
function mulPkTest()   public
    view
    returns (uint256 x, uint256 y, bool success)  {
        
    uint256 scalar = 0xeb94edbc8d5113cc505ebb489d47ade76bc3f02fd02445f7f47fea454faa81ae;
    uint256 xPk = 0x79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798;
    uint256 yPk = 0x483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8;
    
    return mulPk(scalar,yPk,xPk);

}    
    
function mulPk(uint256 scalar, uint256 xPk, uint256 yPk)
    public
    view
    returns (uint256 x, uint256 y, bool success) {
       bytes32 functionSelector = 0xa99aa2f200000000000000000000000000000000000000000000000000000000;
       address to = PRECOMPILE_CONTRACT_ADDR;
       
       assembly {
            let freePtr := mload(0x40)
            mstore(freePtr, functionSelector)
            mstore(add(freePtr, 4), scalar)
            mstore(add(freePtr,36), xPk)
            mstore(add(freePtr,68), yPk)
            
            success := staticcall(gas, to, freePtr,100, freePtr, 64)
            
            x := mload(freePtr)
            y := mload(add(freePtr,32))
        }
        
    }

function s256calarMulTest()   public
    view
    returns (uint256 x, uint256 y, bool success)  {

    uint256 scalar = 0xeb94edbc8d5113cc505ebb489d47ade76bc3f02fd02445f7f47fea454faa81ae;
    uint256 xPk = 0x79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798;
    uint256 yPk = 0x483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8;

    return s256ScalarMul(scalar,yPk,xPk);
}

function s256ScalarMul(uint256 scalar, uint256 xPk, uint256 yPk)
    public
    view
    returns (uint256 x, uint256 y, bool success) {
       address to = 0x43;

       assembly {
            let freePtr := mload(0x40)
            mstore(add(freePtr, 0), scalar)
            mstore(add(freePtr,32), xPk)
            mstore(add(freePtr,64), yPk)

            success := staticcall(gas, to, freePtr,100, freePtr, 64)

            x := mload(freePtr)
            y := mload(add(freePtr,32))
        }

    }


function s256AddTest() public view returns(uint256 retx, uint256 rety,bool success) {
       uint256 x1 = 0x69088a1c79a78b5e66859a5e6594d70c8f12a1ff882d84a05ffdbbcff5a4abcb;
       uint256 y1 = 0x5d4c67c05b0a693fb72b47abf7e0d6381fc722ca45c8bb076e6cb4f9f0912906;

       uint256 x2 = 0xfb4a50e7008341df6390ad3dcd758b1498959bf18369edc335435367088910c6;
       uint256 y2 = 0xe55f58908701c932768c2fd16932f694acd30e21a5f2a4f6242b5f0567696240;

       return s256add(x1,y1,x2,y2);
 }

function s256add(uint256 x1, uint256 y1, uint256 x2,uint256 y2)  public view returns(uint256 retx, uint256 rety,bool success) {

       address to = 0X42;

       assembly {
            let freePtr := mload(0x40)
            mstore(add(freePtr, 0), x1)
            mstore(add(freePtr, 32), y1)
            mstore(add(freePtr, 64), x2)
            mstore(add(freePtr, 96), y2)

            // call ERC20 Token contract transfer function
            success := staticcall(gas,to, freePtr,132, freePtr, 64)

            retx := mload(freePtr)
            rety := mload(add(freePtr,32))
        }

    }

    
}