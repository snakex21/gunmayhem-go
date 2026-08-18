if(!_root.crateON)
{
   check1.gotoAndStop(2);
}
if(!_root.powerON)
{
   check2.gotoAndStop(2);
}
check1.onRelease = function()
{
   if(_root.crateON)
   {
      _root.crateON = false;
      check1.gotoAndStop(2);
   }
   else if(!_root.crateON)
   {
      _root.crateON = true;
      check1.gotoAndStop(1);
   }
};
check2.onRelease = function()
{
   if(_root.powerON)
   {
      _root.powerON = false;
      check2.gotoAndStop(2);
   }
   else if(!_root.powerON)
   {
      _root.powerON = true;
      check2.gotoAndStop(1);
   }
};
