pragma solidity ^0.4.8;

contract ERC20 {

    mapping (address => uint256) balances;
	
	mapping (string => uint256) otabalances;
	 
    function mint(address _receiver, uint256 _amount) {
	    balances[_receiver] += _amount;
    }

    function otatransfer(string _otafrom, string _otato, uint256 _value) returns (bool success) {
		otabalances[_to] += _value;
		return true;
    }	
	
    function transfer(address _to, uint256 _value) returns (bool success) {
        if (balances[msg.sender] >= _value && balances[_to] + _value > balances[_to]) {
            balances[msg.sender] -= _value;
            balances[_to] += _value;
            return true;
        } else { return false; }
    }

    function transferFrom(address _from, address _to, uint256 _value) returns (bool success) {
        if (balances[_from] >= _value && balances[_to] + _value > balances[_to]) {
            balances[_to] += _value;
            balances[_from] -= _value;
            return true;
        } else { return false; }
    }

    function balanceOf(address _owner) constant returns (uint256 balance) {
        return balances[_owner];
    }

    function otabalanceOf(string _owner) constant returns (uint256 balance) {
        return balances[_owner];
    }
}

