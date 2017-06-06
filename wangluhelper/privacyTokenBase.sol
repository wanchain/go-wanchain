pragma solidity ^0.4.0;

contract PrivacyTokenBase {
    bool public mInitialized;
    mapping (address => uint256) public balanceOf;
    mapping (address => bytes)   public keyOf;//key

    function PrivacyTokenBase(){
        mInitialized = false;
    }

    function initAsset(address initialBase, bytes baseKeyBytes, uint256 value) returns (bool success){    
        if(mInitialized == true)
            return false;
            
        mInitialized = true;
        balanceOf[initialBase] = value;
        keyOf[initialBase] = baseKeyBytes;
        return true;
    }

    function privacyTransfer(address tfrom, address tto, bytes keyBytes, uint256 _value, uint8 sigv, bytes32 sigr, bytes32 sigs) returns (string) {
        //if (balanceOf[tto] != 0) throw;
        if(balanceOf[tfrom] < _value) return "hhhh";
        
        // check signature signed by tfrom
        bytes32 sv = uintToBytes(_value);
        bytes32 inputHash = sha3(tfrom, tto, keyBytes, sv);
        address recoveredAddress = ecrecover(inputHash, sigv, sigr, sigs);
        
        if(recoveredAddress != tfrom)
            return "address not matched";
            
        balanceOf[tfrom] -= _value;
        balanceOf[tto] += _value;
        keyOf[tto] = keyBytes;
        return "success";
    }
    
    function uintToBytes(uint256 v) constant returns (bytes32 ret) {
        if (v == 0) {
            ret = '0';
        }
        else {
            while (v > 0) {
                ret = bytes32(uint(ret) / (2 ** 8));
                ret |= bytes32(((v % 10) + 48) * 2 ** (8 * 31));
                v /= 10;
            }
        }
        return ret;
    }    

    //TODO: verify publickey keybytes corresponding to address

    //below functions are used for debug

    function tranferDirect(address tfrom, address tto, uint256 value) {
        balanceOf[tfrom] -= value;
        balanceOf[tto] += value;        
    }

    function signBytes(address tfrom, address tto, bytes keyBytes, uint256 _value) returns (bytes32){
        return sha3(tfrom, tto, keyBytes, uintToBytes(_value));
    }


    function sigCheck(address tfrom, address tto, bytes keyBytes, uint256 value,uint8 sigv, bytes32 sigr, bytes32 sigs) returns (address){
        bytes32 sv = uintToBytes(value);
        bytes32 inputHash = sha3(tfrom, tto, keyBytes, sv);
        return ecrecover(inputHash, sigv, sigr, sigs);
    }    


    function sigCheckByHash(bytes32 hash, uint8 sigv, bytes32 sigr, bytes32 sigs) returns (address){
        return ecrecover(hash, sigv, sigr, sigs);
    }        
}
