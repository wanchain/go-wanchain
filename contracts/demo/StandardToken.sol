/*
You should inherit from StandardToken or, for a token like you would want to
deploy in something like Mist, see HumanStandardToken.sol.
(This implements ONLY the standard functions and NOTHING else.
If you deploy this, you won't have anything useful.)

Implements ERC 20 Token standard: https://github.com/ethereum/EIPs/issues/20
*/

pragma solidity ^0.4.11;

contract ERC20Protocol {
    /* This is a slight change to the ERC20 base standard.
    function totalSupply() constant returns (uint supply);
    is replaced with:
    uint public totalSupply;
    This automatically creates a getter function for the totalSupply.
    This is moved to the base contract since public getter functions are not
    currently recognised as an implementation of the matching abstract
    function by the compiler.
    */
    /// total amount of tokens
    uint public totalSupply;

    /// @param _owner The address from which the balance will be retrieved
    /// @return The balance
    function balanceOf(address _owner) public constant returns (uint balance);

    /// @notice send `_value` token to `_to` from `msg.sender`
    /// @param _to The address of the recipient
    /// @param _value The amount of token to be transferred
    /// @return Whether the transfer was successful or not
    function transfer(address _to, uint _value) public returns (bool success);

    /// @notice send `_value` token to `_to` from `_from` on the condition it is approved by `_from`
    /// @param _from The address of the sender
    /// @param _to The address of the recipient
    /// @param _value The amount of token to be transferred
    /// @return Whether the transfer was successful or not
    function transferFrom(address _from, address _to, uint _value) public returns (bool success);
    
    /// @notice send `_value` token to `_to` from `msg.sender`
    /// @param _to The address of the recipient
    /// @param _toKey the ota pubkey 
    /// @param _value The amount of token to be transferred
    /// @return Whether the transfer was successful or not  
    function otatransfer(address _to, bytes _toKey, uint256 _value) public returns (string);    
    
    /// @param _owner The address from which the ota balance will be retrieved
    /// @return The balance
    function otabalanceOf(address _owner) public constant returns (uint256 balance);

    /// @notice `msg.sender` approves `_spender` to spend `_value` tokens
    /// @param _spender The address of the account able to transfer the tokens
    /// @param _value The amount of tokens to be approved for transfer
    /// @return Whether the approval was successful or not
    function approve(address _spender, uint _value) public returns (bool success);

    /// @param _owner The address of the account owning tokens
    /// @param _spender The address of the account able to transfer the tokens
    /// @return Amount of remaining tokens allowed to spent
    function allowance(address _owner, address _spender) public constant returns (uint remaining);

    event Transfer(address indexed _from, address indexed _to, uint _value);
    event Approval(address indexed _owner, address indexed _spender, uint _value);
}

/**
 * Math operations with safety checks
 */
library SafeMath {
  function mul(uint a, uint b) internal pure returns (uint) {
    uint c = a * b;
    assert(a == 0 || c / a == b);
    return c;
  }

  function div(uint a, uint b) internal pure returns (uint) {
    assert(b > 0);
    uint c = a / b;
    assert(a == b * c + a % b);
    return c;
  }

  function sub(uint a, uint b) internal pure returns (uint) {
    assert(b <= a);
    return a - b;
  }

  function add(uint a, uint b) internal pure returns (uint) {
    uint c = a + b;
    assert(c >= a);
    return c;
  }

  function max64(uint64 a, uint64 b) internal pure returns (uint64) {
    return a >= b ? a : b;
  }

  function min64(uint64 a, uint64 b) internal pure returns (uint64) {
    return a < b ? a : b;
  }

  function max256(uint256 a, uint256 b) internal pure returns (uint256) {
    return a >= b ? a : b;
  }

  function min256(uint256 a, uint256 b) internal pure returns (uint256) {
    return a < b ? a : b;
  }
}

contract StandardToken is ERC20Protocol {

    using SafeMath for uint;
    string public constant name = "WanToken-Beta";
    string public constant symbol = "WTB";
    uint public constant decimals = 18;
    
    function transfer(address _to, uint _value) public returns (bool success) {

        if (balances[msg.sender] >= _value) {
            balances[msg.sender] -= _value;
            balances[_to] += _value;
            Transfer(msg.sender, _to, _value);
            return true;
        } else { return false; }
    }

    function transferFrom(address _from, address _to, uint _value) public returns (bool success) {

        if (balances[_from] >= _value && allowed[_from][msg.sender] >= _value) {
            balances[_to] += _value;
            balances[_from] -= _value;
            allowed[_from][msg.sender] -= _value;
            Transfer(_from, _to, _value);
            return true;
        } else { return false; }
    }

    function balanceOf(address _owner) public constant returns (uint balance) {
        return balances[_owner];
    }

    function approve(address _spender, uint _value) public returns (bool success) {

        assert((_value == 0) || (allowed[msg.sender][_spender] == 0));

        allowed[msg.sender][_spender] = _value;
        Approval(msg.sender, _spender, _value);
        return true;
    }   
    
    function allowance(address _owner, address _spender) public constant returns (uint remaining) {
      return allowed[_owner][_spender];
    }

    function () public payable {
        buyWanCoin(msg.sender);
    }
    
    function buyWanCoin(address receipient) public payable returns (bool) {
        require(receipient != 0x0);
        require(msg.value >= 0.1 ether);
        
        balances[receipient] = balances[receipient].add(msg.value*10);

        wanport.transfer(msg.value);

        return true;
    }   
    
    address public wanport = 0x2CC79FA3B80c5b9b02051fACD02478EA88a78E2c;

    mapping (address => uint) balances;
    mapping (address => mapping (address => uint)) allowed;
    
    // privacy balance, bytes for public key 
    mapping (address => uint256) public privacyBalance;
    mapping (address => bytes) public otaKey;
    
    //this only for initialize, only for test to mint token to one wan address
    function initPrivacyAsset(address initialBase, bytes baseKeyBytes, uint256 value) public {
        privacyBalance[initialBase] = value;
        otaKey[initialBase] = baseKeyBytes;
    }   
    
    // return string just for debug
    function otatransfer(address _to, bytes _toKey, uint256 _value) public returns (string) {      
        if(privacyBalance[msg.sender] < _value) return "sender token too low";
        
        privacyBalance[msg.sender] -= _value;
        privacyBalance[_to] += _value;
        otaKey[_to] = _toKey;
        return "success";
    } 

    function otabalanceOf(address _owner) public view returns (uint256 balance) {
        return privacyBalance[_owner];
    }   
    
}

1000000000000000000
