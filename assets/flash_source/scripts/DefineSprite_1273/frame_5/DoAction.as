if(!_root.crateON)
{
   check1.gotoAndStop(2);
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
