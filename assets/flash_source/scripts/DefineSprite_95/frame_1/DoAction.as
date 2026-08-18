switch(this._name)
{
   case "panel1":
      number = 1;
      break;
   case "panel2":
      number = 2;
      break;
   case "panel3":
      number = 3;
      break;
   case "panel4":
      number = 4;
      break;
   case "panel5":
      number = 5;
      break;
   case "panel6":
      number = 6;
      break;
   case "panel7":
      number = 7;
      break;
   case "panel8":
      number = 8;
      break;
   case "panel9":
      number = 9;
      break;
   case "panel10":
      number = 10;
}
cicon.gotoAndStop(number);
if(_root.savedata3.data.levelarray[number - 1] == 2)
{
   lock.gotoAndStop(3);
}
if(_root.savedata3.data.levelarray[number - 1] == 1)
{
   lock.gotoAndStop(2);
}
if(_root.savedata3.data.levelarray[number - 1] == 0)
{
   this.useHandCursor = false;
}
this.onPress = function()
{
   if(_parent._alpha >= 100 && lock._currentframe != 1 && this.useHandCursor)
   {
      _root.playsound("menu.wav");
      _parent.disableall();
      _parent._parent.gotolevel(number);
   }
};
