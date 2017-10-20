pragma solidity ^0.4.8;

contract ERC20 {

    mapping (address => uint256) balances;
	
	mapping (string => uint256) otabalances;
	
	address public constant MINT_CALLER =  0x0000000000000000000000000000000000000000;								 
	
	modifier onlyMintCaller {
        require(msg.sender == MINT_CALLER);
        _;
    }
	 
	 
    function mint(address _receiver, uint256 _amount) {
	    balances[_receiver] += _amount;		
    }
	
	//this only for initialize, only for test to mint token to one wan address
    function otamint(string _receiver, uint256 _amount) {
	    otabalances[_receiver] += _amount;
		
    }	
	
    function otatransfer(string _from, string _to, uint256 _value) onlyMintCaller returns (bool success) {		
        if (otabalances[_from] >= _value && otabalances[_to] + _value > otabalances[_to]) {
            otabalances[_to] += _value;
            otabalances[_from] -= _value;
            return true;
        } else { 
		   return false; 
		}
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
        return otabalances[_owner];
    }
}


