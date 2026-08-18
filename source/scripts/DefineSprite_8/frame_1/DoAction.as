_alpha = 0;
switch(_name)
{
   case "btn1":
      btnNumber = 1;
      break;
   case "btn2":
      btnNumber = 2;
      break;
   case "btn3":
      btnNumber = 3;
      break;
   case "btn4":
      btnNumber = 4;
      break;
   case "btn5":
      btnNumber = 5;
      break;
   case "btn6":
      btnNumber = 6;
}
this.onRollOver = function()
{
   _parent.targety = _Y;
   if(_parent.slide._alpha < 20)
   {
      _parent.slide._y = _Y;
   }
   if(_parent.slide._currentframe != btnNumber)
   {
      _parent.slide.gotoAndStop(btnNumber);
   }
};
